package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jachin/weread-helper/internal/appcfg"
	"github.com/jachin/weread-helper/internal/config"
	"github.com/jachin/weread-helper/internal/httpapi"
	"github.com/jachin/weread-helper/internal/secret"
	"github.com/jachin/weread-helper/internal/store"
	"github.com/jachin/weread-helper/internal/syncjob"
	"github.com/jachin/weread-helper/internal/weread"
)

func main() {
	cfg := config.Load()

	st, err := store.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	defer st.Close()

	encKey, err := secret.LoadKey(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("加载设置加密密钥失败: %v", err)
	}

	rt, err := appcfg.LoadRuntime(st, encKey, cfg)
	if err != nil {
		log.Fatalf("加载应用设置失败: %v", err)
	}
	if rt.APIKey == "" {
		log.Print("尚未配置 API Key，请打开设置页填写")
	}

	client := weread.New(rt.GatewayURL, rt.APIKey, rt.SkillVersion)
	job := syncjob.New(client, st, rt.SyncInterval)
	job.MaybeStart(false)

	go func() {
		t := time.NewTicker(10 * time.Minute)
		defer t.Stop()
		for range t.C {
			job.MaybeStart(false)
		}
	}()

	srv := httpapi.New(st, job, client, encKey)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://127.0.0.1:5173", "http://localhost:5173"},
		AllowMethods: []string{"GET", "POST", "PUT", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type"},
	}))
	srv.Register(r)

	log.Printf("listening on %s db=%s interval=%s", cfg.ListenAddr, cfg.DatabasePath, rt.SyncInterval)
	if err := http.ListenAndServe(cfg.ListenAddr, r); err != nil {
		log.Fatal(err)
	}
}
