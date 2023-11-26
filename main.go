package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/monitor"
	"github.com/gopcua/opcua/ua"
)

type Config struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
	Name string `json:"name"`
}

var clients []*opcua.Client

func main() {

	ctx := context.Background()

	err := os.Setenv("OPCUA_ENDPOINTS", `[{"ip":"192.168.178.108","name":"Line 1","port":49320},{"ip":"192.168.178.108","name":"Line 2","port":49320}]`)

	if err != nil {
		fmt.Println(err)
	}

	conf := []byte(os.Getenv("OPCUA_ENDPOINTS"))

	var machines []Config

	if err := json.Unmarshal(conf, &machines); err != nil {
		fmt.Println(err)
	}
	fmt.Println(machines)

	wg := sync.WaitGroup{}

	for _, ep := range machines {

		fmt.Println(ep)

		c, err := initOPCClient(ctx, ep.IP, strconv.Itoa(ep.Port))

		if err != nil {
			fmt.Println(err)
			continue
		}

		clients = append(clients, c)

		m, err := monitor.NewNodeMonitor(c)

		if err != nil {
			fmt.Println(err)
			continue
		}
		wg.Add(1)
		go initKeepalive(ctx, m, &wg, ep.Name)

	}

	wg.Wait()
}

func initOPCClient(ctx context.Context, ip string, port string) (*opcua.Client, error) {

	url := fmt.Sprintf("opc.tcp://%s:%s", ip, port)

	endpoints, err := opcua.GetEndpoints(ctx, url)

	if err != nil {
		return nil, err
	}

	ep := opcua.SelectEndpoint(endpoints, "None", ua.MessageSecurityModeFromString("None"))

	opts := []opcua.Option{
		opcua.AuthAnonymous(),
		opcua.SecurityMode(ua.MessageSecurityModeNone),
		opcua.SecurityPolicy("None"),
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
	}

	c, err := opcua.NewClient(url, opts...)

	if err != nil {
		return nil, err
	}

	if err := c.Connect(ctx); err != nil {
		return nil, err
	}

	return c, nil
}

func initKeepalive(ctx context.Context, m *monitor.NodeMonitor, wg *sync.WaitGroup, name string) {

	sub, err := m.Subscribe(ctx, &opcua.SubscriptionParameters{Interval: 10 * time.Second, Priority: 1}, func(s *monitor.Subscription, dcm *monitor.DataChangeMessage) {
		if dcm.Error != nil {
			fmt.Println(dcm.Error)

		} else {
			fmt.Printf("Machine: %s- Is Online \n", name)
		}

	}, "i=2258")

	if err != nil {
		wg.Done()
	}

	fmt.Println("Starting Keepalive")

	defer cleanup(ctx, sub, wg)

	<-ctx.Done()
}

func cleanup(ctx context.Context, sub *monitor.Subscription, wg *sync.WaitGroup) {
	fmt.Println("Cleanup sub")
	sub.Unsubscribe(ctx)
	wg.Done()
}
