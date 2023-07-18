package main

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/google/uuid"
	"github.com/twitchscience/kinsumer"
)

type Consumer struct {
	kinsumer *kinsumer.Kinsumer
	logger   *log.Logger
	wg       sync.WaitGroup
	recordCh chan []byte
	errCh    chan error
	Name     string
}

type ScanFunc func([]byte) error

func NewConsumer() (*Consumer, error) {
	streamName := os.Getenv("STREAM_NAME")
	stackName := os.Getenv("STACK_NAME")
	logger := log.New(os.Stdout, "Consumer: ", log.Lshortfile)
	config := kinsumer.NewConfig().WithStats(NewStatsReceiver())
	session := session.Must(session.NewSession())
	logger.Println("streamName: ", streamName)
	name := uuid.New().String()
	// ここではCDKスタック名をconsumer nameとして利用
	kinsumer, err := kinsumer.NewWithSession(session, streamName, stackName, name, config)

	consumer := &Consumer{
		kinsumer: kinsumer,
		logger:   logger,
		wg:       sync.WaitGroup{},
		Name:     name,
	}
	return consumer, err

}

func (c *Consumer) Run() error {
	err := c.kinsumer.Run()
	if err != nil {
		c.logger.Println("Error running kinsumer: ", err)
	}
	return err
}

func (c *Consumer) Stop() {
	// kinsumer.Stopのラッパー
	c.logger.Println("Stopping consumer...")
	c.kinsumer.Stop()
	c.logger.Println("stopped consumer")
}

func (c *Consumer) Scan(ctx context.Context, fn ScanFunc) {
	// シャードのスキャンを開始する
	c.errCh = make(chan error)
	c.recordCh = make(chan []byte)

	// kinsumer.Next()のgoroutine
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			// Next()はブロック関数であることに留意
			record, err := c.kinsumer.Next()
			if err != nil {
				c.errCh <- err
				return
			}
			if record != nil {
				c.recordCh <- record
			} else {
				c.logger.Println("record is nil")
				return
			}
		}
	}()

	// scanFuncのgoroutine
	// kinsumerがcontextに対応していないため、ここでcontextを処理する
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-ctx.Done():
				// contextがキャンセルされたらkinsumerを停止
				c.Stop()
				return
			case err := <-c.errCh:
				// kinsumer.Next()でエラーが発生したらkinsumerを停止
				c.logger.Println("Error on kinsumer.Next: ", err)
				c.Stop()
				return
			case record := <-c.recordCh:
				// kinsumer.Next()で取得したレコードをscanFuncに渡す
				if err := fn(record); err != nil {
					c.logger.Println("Error in scanFunc: ", err)
				}
			}
		}
	}()
}
