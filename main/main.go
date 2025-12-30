package main

// import (
// 	"context"
// 	"fmt"
// 	"time"

// 	"github.com/Amirhos-esm/startable"
// )

// type Counter struct {
// 	startable.Startable
// }

// func main() {
// 	counter := Counter{}
// 	counter.SetPreStartCallback(func() {
// 		fmt.Println("SetPreStartCallback")
// 	})
// 	counter.SetPostStartCallback(func(a any) {
// 		fmt.Println("SetPostStartCallback.ret: ", a)
// 	})
// 	counter.SetStartFunction(func(ctx context.Context) any {

// 		for range 5 {

// 			select {
// 			case <-time.After(1 * time.Second):
// 				fmt.Println("after 1 seconds")
// 			case <-ctx.Done():
// 				fmt.Println("request to close!")
// 				time.Sleep(2 * time.Second)
// 				fmt.Println("closed")
// 				return 0
// 			}

// 		}
// 		return -1
// 	})
// 	go func() {
// 		for {
// 			time.Sleep(1 * time.Second)
// 			fmt.Println("isStarted,", counter.IsStarted())
// 		}
// 	}()
// 	fmt.Println("start,", counter.Start())
// 	time.Sleep(3 * time.Second) // let it not to run
// 	fmt.Println("stop,", counter.Stop())
// }
