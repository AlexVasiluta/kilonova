package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	common "github.com/KiloProjects/Kilonova/common"
	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/grader/judge"
	"github.com/KiloProjects/Kilonova/kndb"
	"github.com/KiloProjects/Kilonova/server"
	"github.com/KiloProjects/Kilonova/web"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"gorm.io/driver/postgres"
	_ "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// go:generate pkger

var (
	masterDB *gorm.DB
	config   *common.Config
	manager  *datamanager.StorageManager
	db       *kndb.DB

	dataDir    = flag.String("data", "/data", "Data directory")
	configFile = flag.String("config", "/app/config.json", "Config directory")
)

func main() {
	flag.Parse()

	config, err := readConfig()
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("Read config")

	fmt.Println("Trying to connect to DB until it works")
	for {
		dsn := "sslmode=disable host=db user=kilonova password=kn_password dbname=kilonova"
		masterDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
	}
	fmt.Println("Connected to DB")

	db = kndb.New(masterDB)

	db.AutoMigrate()

	db.DB.Logger = logger.Default.LogMode(logger.Warn)

	manager = datamanager.NewManager(*dataDir)

	// Initialize router
	r := chi.NewRouter()
	corsConfig := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})
	r.Use(corsConfig.Handler)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.Timeout(20 * time.Second))
	r.Use(middleware.Logger)
	r.Use(middleware.RealIP)

	// Setup context
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize components
	API := server.NewAPI(ctx, db, config, manager)
	grader := judge.NewGrader(ctx, db.DB, manager)

	err = grader.NewManager(2)
	if err != nil {
		panic(err)
	}

	r.Mount("/api", API.GetRouter())
	r.Mount("/", web.NewWeb(manager, db).GetRouter())

	// TODO: Find out why memory usage is higher than on pbinfo.ro (which also uses `isolate`) for the same program
	grader.Start()
	defer grader.Shutdown()

	// for graceful setup and shutdown
	server := &http.Server{Addr: "0.0.0.0:8080", Handler: r}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			fmt.Println(err)
		}
	}()
	fmt.Println("Successfully started")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop
	cancel()
	fmt.Println("Shutting Down")
	if err := server.Shutdown(ctx); err != nil {
		fmt.Println(err)
	}
}

func readConfig() (*common.Config, error) {
	data, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return nil, err
	}
	var config common.Config
	json.Unmarshal(data, &config)
	return &config, nil
}
