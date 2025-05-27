package metric

import (
	"github.com/shirou/gopsutil/v4/net"
)

func GetNetInformation() (MetricsSlice, []CustomErr) {
	var netData MetricsSlice
	var netErrors []CustomErr
	netIOCounters, err := net.IOCounters(true)
	if err != nil {
		netErrors = append(netErrors, CustomErr{
			Metric: []string{"net"},
			Error:  err.Error(),
		})
		return nil, netErrors
	}

	for _, v := range netIOCounters {
		netData = append(netData, NetData{
			Name:        v.Name,
			BytesSent:   v.BytesSent,
			BytesRecv:   v.BytesRecv,
			PacketsSent: v.PacketsSent,
			PacketsRecv: v.PacketsRecv,
			ErrIn:       v.Errin,
			ErrOut:      v.Errout,
			DropIn:      v.Dropin,
			DropOut:     v.Dropout,
			FIFOIn:      v.Fifoin,
			FIFOOut:     v.Fifoout,
		})
	}

	return netData, nil
}
