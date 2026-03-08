package jobs

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/internal/db"
	"github.com/srad/mediasink/internal/util"
)

var (
	cancelMetrics context.CancelFunc
)

func StartMetrics(networkDev string) {
	ctx, c := context.WithCancel(context.Background())
	cancelMetrics = c
	go trackCPU(ctx)
	go trackNetwork(ctx, networkDev)
}

func StopMetrics() {
	cancelMetrics()
}

func trackCPU(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Infoln("[trackCPU] stopped")
			return
		default:
			// sleeps automatically
			cpu, err := util.CPUUsage(30)
			if err != nil {
				log.Errorf("[trackCPU] Error reasing cpu: %s", err)
				return
			}

			if err := db.DB.Model(&util.CPULoad{}).Create(cpu.LoadCPU).Error; err != nil {
				log.Errorf("[trackCPU] Error saving metric: %s", err)
			}
		}
	}
}

func trackNetwork(ctx context.Context, networkDev string) {
	for {
		select {
		case <-ctx.Done():
			log.Infoln("[trackNetwork] stopped")
			return
		default:
			netInfo, err := util.NetMeasure(networkDev, 15)
			if err != nil {
				log.Errorln("[trackNetwork] stopped")
				return
			}
			if err := db.DB.Model(&util.NetInfo{}).Create(netInfo).Error; err != nil {
				log.Errorf("[trackCPU] Error saving metric: %s", err)
			}
		}
	}
}
