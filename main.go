package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/acoshift/configfile"
	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"

	"github.com/xkamail/smartfan/pkg/miio"
)

var cfg = configfile.NewEnvReader()

func main() {
	configfile.LoadDotEnv()

	m, err := miio.New(cfg.String("ip"), cfg.String("device_token"))
	if err != nil {
		panic(err)
	}
	a := accessory.NewFan(accessory.Info{
		Name:  "Smart Fan",
		Model: "xiaomi.fan.p45",
	})
	a.Fan.On.OnValueRemoteUpdate(func(v bool) {
		m.SetPower(v)
	})

	fanSpeed := characteristic.NewRotationSpeed()
	a.Fan.AddC(fanSpeed.C)
	fanSpeed.OnValueRemoteUpdate(func(v float64) {
		m.SetLevel(int(v))
	})

	fanSwing := characteristic.NewSwingMode()
	a.Fan.AddC(fanSwing.C)
	fanSwing.OnValueRemoteUpdate(func(v int) {
		log.Println("Fan Swing: ", v)
		m.SetSwing(v == 1)
	})

	fanDirection := characteristic.NewRotationDirection()
	a.Fan.AddC(fanDirection.C)
	fanDirection.OnValueRemoteUpdate(func(v int) {
		m.SetSwing(v == 1)
	})

	// update state from mi

	m.OnUpdate(func(level int, swing bool, power bool) {
		a.Fan.On.SetValue(power)
		fanSpeed.SetValue(float64(level))
		dir := 0
		if swing {
			dir = 1
		}
		fanDirection.SetValue(dir)
		fanSwing.SetValue(dir)
	})

	// Store the data in the "./db" directory.
	fs := hap.NewFsStore("./db")

	// Create the hap server.
	server, err := hap.NewServer(fs, a.A)
	if err != nil {
		// stop if an error happens
		log.Panic(err)
	}

	// Setup a listener for interrupts and SIGTERM signals
	// to stop the server.
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-c
		// Stop delivering signals.
		signal.Stop(c)
		// Cancel the context to stop the server.
		cancel()
	}()

	server.Pin = "00102003"

	// Run the server.
	server.ListenAndServe(ctx)
}
