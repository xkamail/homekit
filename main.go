package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/acoshift/configfile"
	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"

	"github.com/xkamail/smartfan/pkg/miio"
)

var cfg = configfile.NewEnvReader()

func main() {
	configfile.LoadDotEnv()

	fan, fanDevice, err := newSmartFan(cfg.String("fan_ip"), cfg.String("fan_token"))
	if err != nil {
		log.Panic(err)
		return
	}

	go func() {
		tick := time.NewTicker(time.Second * 5)
		defer tick.Stop()
		fanDevice.PollStatus(tick.C)
	}()

	// Store the data in the "./db" directory.
	fs := hap.NewFsStore("./db")

	//a3 := accessory.NewCooler(accessory.Info{
	//	Name:  "Cool",
	//	Model: "zhimi.aircondition.ma2",
	//})
	//a3.Cooler.CoolingThresholdTemperature.SetMinValue(10)
	//a3.Cooler.CoolingThresholdTemperature.SetValue(25)
	//a3.Cooler.CurrentTemperature.SetValue(25)
	//a3.Cooler.TargetHeaterCoolerState.ValidVals = []int{
	//	//characteristic.TargetHeaterCoolerStateAuto,
	//	characteristic.TargetHeaterCoolerStateCool,
	//	//characteristic.TargetHeaterCoolerStateHeat,
	//}
	//h := characteristic.NewHeatingThresholdTemperature()
	//a3.Cooler.AddC(h.C)
	//
	//a3.Cooler.Active.OnValueRemoteUpdate(func(v int) {
	//	log.Println("Cooler Active: ", v)
	//})
	//a3.Cooler.CurrentHeaterCoolerState.OnValueRemoteUpdate(func(v int) {
	//	log.Println("Cooler Current: ", v)
	//})
	//a3.Cooler.TargetHeaterCoolerState.OnValueRemoteUpdate(func(v int) {
	//	log.Println("Cooler Target: ", v)
	//})
	//a3.Cooler.CurrentTemperature.OnValueRemoteUpdate(func(v float64) {
	//	log.Println("Cooler Current Temperature: ", v)
	//})
	//a3.Cooler.CoolingThresholdTemperature.OnValueRemoteUpdate(func(v float64) {
	//	log.Println("Cooler Cooling Threshold Temperature: ", v)
	//})

	// Create the hap server.
	server, err := hap.NewServer(fs, fan)
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

func newSmartFan(ip string, token string) (*accessory.A, *miio.SmartFan, error) {
	m, err := miio.NewSmartFan(ip, token)
	if err != nil {
		return nil, nil, err
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
		m.SetSwing(v == 1)
	})

	fanDirection := characteristic.NewRotationDirection()
	a.Fan.AddC(fanDirection.C)

	fanDirection.OnValueRemoteUpdate(func(v int) {
		m.SetSwing(v == 1)
	})

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
	return a.A, m, nil
}

func newAirPurifier(ip, token string) (*accessory.A, error) {
	m, err := miio.NewAirPurifier(ip, token)
	if err != nil {
		return nil, err
	}
	_ = m

	a := accessory.NewAirPurifier(accessory.Info{
		Name:  "Air Purifier C3",
		Model: "zhimi.airpurifier.mb4",
	})

	err = a.AirPurifier.Active.SetValue(characteristic.ActiveInactive)
	if err != nil {
		return nil, err
	}
	a.AirPurifier.Active.OnValueRemoteUpdate(func(v int) {
		if v == characteristic.ActiveActive {
			v = 2
		}
		m.SetMode(v)
	})

	airPurifierSensor := characteristic.NewAirQuality()
	airPurifierSensor.SetValue(characteristic.AirQualityExcellent)
	a.AirPurifier.AddC(airPurifierSensor.C)

	filterCondition := characteristic.NewFilterLifeLevel()
	filterCondition.SetValue(100)
	a.AirPurifier.AddC(filterCondition.C)

	a.AirPurifier.CurrentAirPurifierState.OnValueRemoteUpdate(func(v int) {
		log.Println("Air Purifier Current: ", v)
	})
	a.AirPurifier.TargetAirPurifierState.OnValueRemoteUpdate(func(v int) {
		log.Println("Air Purifier Target: ", v)
	})

	a.AirPurifier.CurrentAirPurifierState.SetValue(characteristic.CurrentAirPurifierStatePurifyingAir)

	m.OnUpdate(func(mode int, quality int) {
		if quality < 3 {
			airPurifierSensor.SetValue(characteristic.AirQualityExcellent)
		} else if quality < 6 {
			airPurifierSensor.SetValue(characteristic.AirQualityGood)
		} else if quality < 9 {
			airPurifierSensor.SetValue(characteristic.AirQualityFair)
		} else {
			airPurifierSensor.SetValue(characteristic.AirQualityPoor)
		}
		if mode == 0 {
			a.AirPurifier.TargetAirPurifierState.SetValue(characteristic.TargetAirPurifierStateAuto)
			a.AirPurifier.Active.SetValue(characteristic.ActiveInactive)
		}
		if mode == 2 {
			a.AirPurifier.TargetAirPurifierState.SetValue(characteristic.TargetAirPurifierStateManual)
			a.AirPurifier.Active.SetValue(characteristic.ActiveActive)
		}
	})
	return a.A, nil
}
