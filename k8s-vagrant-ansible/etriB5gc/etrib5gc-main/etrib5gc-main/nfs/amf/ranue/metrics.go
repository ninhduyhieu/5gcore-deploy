package ranue

import (
	"time"
)

type RanUeMetrics struct {
	createdTime     time.Time
	lastRegTime     time.Time
	lastRegDuration int64 //time taken to finish registration
}

func createRanUeMetrics() *RanUeMetrics {
	return &RanUeMetrics{
		createdTime: time.Now(),
	}
}

//TODO: copy metrics from source RanUe in handover
func copyRanUeMetrics(m *RanUeMetrics) *RanUeMetrics {
	return &RanUeMetrics{
		createdTime: time.Now(),
	}
}

func (m *RanUeMetrics) regStart(t time.Time) {
	m.lastRegTime = t
}

func (m *RanUeMetrics) regComplete() {
	m.lastRegDuration = time.Now().Sub(m.lastRegTime).Milliseconds()
}
