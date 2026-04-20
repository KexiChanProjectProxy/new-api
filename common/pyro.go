package common

import (
	"runtime"

	"github.com/grafana/pyroscope-go"
)

func StartPyroScope() error {
	pyroscopeUrl := ""
	pyroscopeAppName := "new-api"
	pyroscopeBasicAuthUser := ""
	pyroscopeBasicAuthPassword := ""
	pyroscopeHostname := "new-api"
	mutexRate := 5
	blockRate := 5

	if startupCfg := GetStartupConfig(); startupCfg != nil {
		pyroscopeUrl = startupCfg.Observability.Pyroscope.URL
		pyroscopeAppName = startupCfg.Observability.Pyroscope.AppName
		pyroscopeBasicAuthUser = startupCfg.Observability.Pyroscope.BasicAuthUser
		pyroscopeBasicAuthPassword = startupCfg.Observability.Pyroscope.BasicAuthPassword
		pyroscopeHostname = startupCfg.Observability.Pyroscope.Hostname
		mutexRate = startupCfg.Observability.Pyroscope.MutexRate
		blockRate = startupCfg.Observability.Pyroscope.BlockRate
	}

	if pyroscopeUrl == "" {
		return nil
	}

	runtime.SetMutexProfileFraction(mutexRate)
	runtime.SetBlockProfileRate(blockRate)

	_, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: pyroscopeAppName,

		ServerAddress:     pyroscopeUrl,
		BasicAuthUser:     pyroscopeBasicAuthUser,
		BasicAuthPassword: pyroscopeBasicAuthPassword,

		Logger: nil,

		Tags: map[string]string{"hostname": pyroscopeHostname},

		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,

			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})
	if err != nil {
		return err
	}
	return nil
}
