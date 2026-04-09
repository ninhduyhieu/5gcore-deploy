package sm

import (
	"fmt"
	"gonum.org/v1/gonum/stat"
	"os"
	"time"
)

type Experiment struct {
	createdTime   time.Time
	activatedTime time.Time
}

func (exp *Experiment) duration() int64 {
	return int64(exp.activatedTime.Sub(exp.createdTime))
}

func LogExpResults(f *os.File) {
	if _pool == nil {
		return
	}
	smList := _pool.getSmContexts()
	var dur float64
	var v []float64
	for _, smCtx := range smList {
		dur = float64(smCtx.exp.duration())
		if dur > 0 {
			v = append(v, dur)
		}
		fmt.Fprintf(f, "%s-%d\t%f\n", smCtx.supi, smCtx.id, dur)
	}
	m, std := stat.MeanStdDev(v, nil)
	fmt.Fprintf(f, "Mean = %f, Std = %f", m, std)
}
