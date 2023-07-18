package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	totalNumRecords int
)

func main() {
	// health check server
	port := os.Getenv("PORT")
	server := &http.Server{Addr: fmt.Sprintf(":%s", port), Handler: http.HandlerFunc(handler)}
	// goroutineで起動
	go func() {
		log.Println("Health check server listening on port: ", port)
		if err := server.ListenAndServe(); err != nil {
			log.Printf("shut down: %v", err)
		}
	}()

	// kinsumerをラップしたConsumerを作成
	consumer, err := NewConsumer()
	if err != nil {
		log.Fatalln("Error creating consumer: ", err)
	}
	// kinsumerを起動
	if err := consumer.Run(); err != nil {
		log.Fatalln("Error running consumer: ", err)
	}

	// graceful shutdown用
	ctx, cancel := context.WithCancel(context.Background())

	signals := make(chan os.Signal, 1)
	// Ctrl+C, kill, SIGTERMのシグナルを受け取る
	// ※ECSはSIGTERMを送信してくる
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	// 上記signal受信用のgoroutine
	go func() {
		<-signals
		log.Println("Received signal, shutting down...")
		// health check serverをシャットダウン
		if err := shutdown(ctx, server); err != nil {
			log.Printf("Failed to shutdown: %v", err)
		}
		// kinsumerをシャットダウン
		cancel()
	}()

	// 最後に処理したレコード数をプリント
	defer func() {
		log.Println("Total number of records: ", totalNumRecords)
	}()

	// レコード数を定期的にプリント
	go reportNumRecords(ctx, consumer)
	// kinsumerスキャン開始
	consumer.Scan(
		ctx,
		scanFunc, // スキャンした時の処理
	)
	// kinsumerのすべての処理が完了するまで待機
	// ※Scan()のgoroutineではcancel時の処理を記述しているため、これがないとgraceful shutdownできない
	consumer.wg.Wait()
}

func scanFunc(record []byte) error {
	// スキャンしたときの処理
	// ここではprintするだけ
	log.Println("record: ", string(record))
	// レコード数をインクリメント
	// ※ロックした方が良いかも
	totalNumRecords++
	return nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

func shutdown(ctx context.Context, server *http.Server) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil && err.Error() != "http: Server closed" {
		log.Println("err is not nil")
		return err
	}
	return nil
}

func reportNumRecords(ctx context.Context, consumer *Consumer) {
	// レコード数を定期的にプリント
	ticker := time.NewTicker(3 * time.Second) // 3秒ごとにプリント
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.Printf("Total number of records of %s: %d", consumer.Name, totalNumRecords)
		}
	}
}
