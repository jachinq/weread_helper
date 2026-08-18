use std::io::{Read, Write};
use std::net::TcpListener;
use std::path::{Path, PathBuf};
use std::sync::Mutex;
use std::time::Duration;
use std::{fs, thread};

use tauri::menu::{CheckMenuItem, Menu, MenuEvent, MenuItem, PredefinedMenuItem};
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri::{AppHandle, Manager, RunEvent, WebviewUrl, WebviewWindowBuilder, WindowEvent};
use tauri_plugin_autostart::MacosLauncher;
use tauri_plugin_shell::process::CommandChild;
use tauri_plugin_shell::ShellExt;

struct SidecarState(Mutex<Option<CommandChild>>);
struct AutostartMenu(CheckMenuItem<tauri::Wry>);

fn portable_root() -> PathBuf {
    let exe = std::env::current_exe().unwrap_or_else(|_| PathBuf::from("."));
    exe.parent()
        .map(Path::to_path_buf)
        .unwrap_or_else(|| PathBuf::from("."))
}

fn data_dir() -> PathBuf {
    portable_root().join("data")
}

fn launched_from_autostart() -> bool {
    std::env::args().any(|a| a == "--autostart")
}

fn pick_port() -> Result<u16, String> {
    let listener = TcpListener::bind("127.0.0.1:0").map_err(|e| e.to_string())?;
    let port = listener.local_addr().map_err(|e| e.to_string())?.port();
    drop(listener);
    Ok(port)
}

fn resolve_web_dir(app: &AppHandle) -> PathBuf {
    let candidates = [
        app.path().resource_dir().ok().map(|p| p.join("web")),
        app.path()
            .resource_dir()
            .ok()
            .map(|p| p.join("resources").join("web")),
        Some(portable_root().join("resources").join("web")),
        Some(PathBuf::from("resources/web")),
    ];
    for p in candidates.into_iter().flatten() {
        if p.join("index.html").is_file() {
            return p;
        }
    }
    app.path()
        .resource_dir()
        .map(|p| p.join("web"))
        .unwrap_or_else(|_| portable_root().join("resources").join("web"))
}

fn health_ok(port: u16) -> bool {
    let mut stream = match std::net::TcpStream::connect(("127.0.0.1", port)) {
        Ok(s) => s,
        Err(_) => return false,
    };
    let _ = stream.set_read_timeout(Some(Duration::from_secs(1)));
    let _ = stream.set_write_timeout(Some(Duration::from_secs(1)));
    let req = format!(
        "GET /api/health HTTP/1.1\r\nHost: 127.0.0.1:{port}\r\nConnection: close\r\n\r\n"
    );
    if stream.write_all(req.as_bytes()).is_err() {
        return false;
    }
    let mut buf = String::new();
    let _ = stream.read_to_string(&mut buf);
    buf.contains("200") && buf.contains("ok")
}

fn wait_health(port: u16) -> bool {
    for _ in 0..80 {
        if health_ok(port) {
            return true;
        }
        thread::sleep(Duration::from_millis(125));
    }
    false
}

fn show_main(app: &AppHandle) {
    if let Some(win) = app.get_webview_window("main") {
        let _ = win.unminimize();
        let _ = win.show();
        let _ = win.set_focus();
    }
}

fn kill_sidecar(app: &AppHandle) {
    if let Some(state) = app.try_state::<SidecarState>() {
        if let Ok(mut guard) = state.0.lock() {
            if let Some(child) = guard.take() {
                let _ = child.kill();
            }
        }
    }
}

fn spawn_sidecar(app: &AppHandle, port: u16) -> Result<(), String> {
    let data = data_dir();
    fs::create_dir_all(&data).map_err(|e| format!("创建数据目录失败: {e}"))?;
    let db = data.join("weread.db");
    let web = resolve_web_dir(app);
    let listen = format!("127.0.0.1:{port}");

    let sidecar = app
        .shell()
        .sidecar("weread-helper")
        .map_err(|e| e.to_string())?
        .env("GIN_MODE", "release")
        .env("LISTEN_ADDR", &listen)
        .env("DATABASE_PATH", db.to_string_lossy().as_ref())
        .env("WEB_DIR", web.to_string_lossy().as_ref())
        .current_dir(portable_root());

    let (_rx, child) = sidecar.spawn().map_err(|e| format!("启动后端失败: {e}"))?;
    *app.state::<SidecarState>()
        .0
        .lock()
        .map_err(|e| e.to_string())? = Some(child);
    Ok(())
}

fn on_menu(app: &AppHandle, event: MenuEvent) {
    match event.id().as_ref() {
        "show" => show_main(app),
        "quit" => {
            kill_sidecar(app);
            app.exit(0);
        }
        "autostart" => {
            use tauri_plugin_autostart::ManagerExt;
            let mgr = app.autolaunch();
            let enabled = mgr.is_enabled().unwrap_or(false);
            let _ = if enabled {
                mgr.disable()
            } else {
                mgr.enable()
            };
            if let Some(item) = app.try_state::<AutostartMenu>() {
                let _ = item.0.set_checked(mgr.is_enabled().unwrap_or(false));
            }
        }
        _ => {}
    }
}

fn setup_tray(app: &AppHandle) -> Result<CheckMenuItem<tauri::Wry>, Box<dyn std::error::Error>> {
    use tauri_plugin_autostart::ManagerExt;
    let autostart_on = app.autolaunch().is_enabled().unwrap_or(false);

    let show = MenuItem::with_id(app, "show", "显示主窗口", true, None::<&str>)?;
    let autostart = CheckMenuItem::with_id(
        app,
        "autostart",
        "开机自启",
        true,
        autostart_on,
        None::<&str>,
    )?;
    let quit = MenuItem::with_id(app, "quit", "退出", true, None::<&str>)?;
    let sep = PredefinedMenuItem::separator(app)?;
    let menu = Menu::with_items(app, &[&show, &autostart, &sep, &quit])?;

    TrayIconBuilder::new()
        .icon(app.default_window_icon().cloned().ok_or("缺少窗口图标")?)
        .tooltip("微信读书助手")
        .menu(&menu)
        .show_menu_on_left_click(true)
        .on_menu_event(|app, event| on_menu(app, event))
        .on_tray_icon_event(|tray, event| {
            if let TrayIconEvent::Click {
                button: MouseButton::Left,
                button_state: MouseButtonState::Up,
                ..
            } = event
            {
                show_main(tray.app_handle());
            }
        })
        .build(app)?;
    Ok(autostart)
}

pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_autostart::init(
            MacosLauncher::LaunchAgent,
            Some(vec!["--autostart".into()]),
        ))
        .plugin(tauri_plugin_single_instance::init(|app, _argv, _cwd| {
            show_main(app);
        }))
        .manage(SidecarState(Mutex::new(None)))
        .setup(|app| {
            let webview_dir = data_dir().join("webview");
            fs::create_dir_all(&webview_dir)?;

            let autostart_item = setup_tray(app.handle())?;
            app.manage(AutostartMenu(autostart_item));

            let port = pick_port()?;
            spawn_sidecar(app.handle(), port)?;

            let win = WebviewWindowBuilder::new(app, "main", WebviewUrl::App("index.html".into()))
                .title("微信读书助手")
                .inner_size(1120.0, 780.0)
                .data_directory(webview_dir)
                .visible(!launched_from_autostart())
                .build()?;

            let handle = app.handle().clone();
            let hide_on_boot = launched_from_autostart();
            tauri::async_runtime::spawn(async move {
                let ok = tauri::async_runtime::spawn_blocking(move || wait_health(port))
                    .await
                    .unwrap_or(false);
                if let Some(win) = handle.get_webview_window("main") {
                    if ok {
                        if let Ok(url) = format!("http://127.0.0.1:{port}/").parse() {
                            let _ = win.navigate(url);
                        }
                        if !hide_on_boot {
                            let _ = win.show();
                        }
                    }
                }
            });

            let _ = win;
            Ok(())
        })
        .on_window_event(|window, event| {
            if let WindowEvent::CloseRequested { api, .. } = event {
                api.prevent_close();
                let _ = window.hide();
            }
        })
        .build(tauri::generate_context!())
        .expect("启动桌面应用失败")
        .run(|app, event| {
            if let RunEvent::Exit = event {
                kill_sidecar(app);
            }
        });
}
