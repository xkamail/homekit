package main

import (
	"context"
	"log"
	"log/slog"
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

	a2 := accessory.NewAirPurifier(accessory.Info{
		Name:  "Air Purifier",
		Model: "zhimi.airpurifier.mb4",
	})

	a2.AirPurifier.Active.OnValueRemoteUpdate(func(v int) {
		log.Println("Air Purifier Active: ", v)
	})
	airPurifierSensor := characteristic.NewAirQuality()
	airPurifierSensor.SetValue(1)
	a2.AirPurifier.AddC(airPurifierSensor.C)

	filterCondition := characteristic.NewFilterLifeLevel()
	filterCondition.SetValue(1)
	a2.AirPurifier.AddC(filterCondition.C)

	a2.AirPurifier.CurrentAirPurifierState.OnValueRemoteUpdate(func(v int) {
		log.Println("Air Purifier Current: ", v)
	})

	a2.AirPurifier.CurrentAirPurifierState.SetValue(1)
	a2.AirPurifier.TargetAirPurifierState.SetValue(1)

	a3 := accessory.NewCooler(accessory.Info{
		Name:  "Cool",
		Model: "zhimi.aircondition.ma2",
	})
	a3.Cooler.CoolingThresholdTemperature.SetMinValue(10)
	a3.Cooler.CoolingThresholdTemperature.SetValue(25)
	a3.Cooler.CurrentTemperature.SetValue(25)
	a3.Cooler.TargetHeaterCoolerState.ValidVals = []int{
		//characteristic.TargetHeaterCoolerStateAuto,
		characteristic.TargetHeaterCoolerStateCool,
		//characteristic.TargetHeaterCoolerStateHeat,
	}
	h := characteristic.NewHeatingThresholdTemperature()
	a3.Cooler.AddC(h.C)

	a3.Cooler.Active.OnValueRemoteUpdate(func(v int) {
		log.Println("Cooler Active: ", v)
	})
	a3.Cooler.CurrentHeaterCoolerState.OnValueRemoteUpdate(func(v int) {
		log.Println("Cooler Current: ", v)
	})
	a3.Cooler.TargetHeaterCoolerState.OnValueRemoteUpdate(func(v int) {
		log.Println("Cooler Target: ", v)
	})
	a3.Cooler.CurrentTemperature.OnValueRemoteUpdate(func(v float64) {
		log.Println("Cooler Current Temperature: ", v)
	})
	a3.Cooler.CoolingThresholdTemperature.OnValueRemoteUpdate(func(v float64) {
		log.Println("Cooler Cooling Threshold Temperature: ", v)
	})

	// Create the hap server.
	server, err := hap.NewServer(fs, a.A, a2.A, a3.A)
	if err != nil {
		// stop if an error happens
		log.Panic(err)
	}
	// 00102003
	server.Pin = "00002003"

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
		slog.Info("Stopping server...")
	}()

	// Run the server.
	slog.Info("Starting server...")
	server.ListenAndServe(ctx)
}
