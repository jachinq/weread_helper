package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jachin/weread-helper/internal/config"
	"github.com/jachin/weread-helper/internal/httpapi"
	"github.com/jachin/weread-helper/internal/store"
	"github.com/jachin/weread-helper/internal/syncjob"
	"github.com/jachin/weread-helper/internal/weread"
)

func main() {
	cfg := config.Load()
	if cfg.APIKey == "" {
		log.Fatal("请设置 WEREAD_API_KEY（见 .env.example）")
	}

	st, err := store.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	defer st.Close()

	client := weread.New(cfg.GatewayURL, cfg.APIKey, cfg.SkillVersion)
	job := syncjob.New(client, st, cfg.SyncInterval)
	job.MaybeStart(false)

	go func() {
		t := time.NewTicker(10 * time.Minute)
		defer t.Stop()
		for range t.C {
			job.MaybeStart(false)
		}
	}()

	srv := httpapi.New(st, job)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://127.0.0.1:5173", "http://localhost:5173"},
		AllowMethods: []string{"GET", "POST", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type"},
	}))
	srv.Register(r)

	log.Printf("listening on %s db=%s interval=%s", cfg.ListenAddr, cfg.DatabasePath, cfg.SyncInterval)
	if err := http.ListenAndServe(cfg.ListenAddr, r); err != nil {
		log.Fatal(err)
	}
}
