package forwarder

import (
	"etrib5gc/nfs/upf/report"
	"github.com/reogac/pfcp/ie"
)

type Empty struct{}

func (Empty) Close() {
}

func (Empty) CreatePDR(uint64, *ie.CreatePDR) error {
	return nil
}

func (Empty) UpdatePDR(uint64, *ie.UpdatePDR) error {
	return nil
}

func (Empty) RemovePDR(uint64, *ie.RemovePDR) error {
	return nil
}

func (Empty) CreateFAR(uint64, *ie.CreateFAR) error {
	return nil
}

func (Empty) UpdateFAR(uint64, *ie.UpdateFAR) error {
	return nil
}

func (Empty) RemoveFAR(uint64, *ie.RemoveFAR) error {
	return nil
}

func (Empty) CreateQER(uint64, *ie.CreateQER) error {
	return nil
}

func (Empty) UpdateQER(uint64, *ie.UpdateQER) error {
	return nil
}

func (Empty) RemoveQER(uint64, *ie.RemoveQER) error {
	return nil
}

func (Empty) CreateURR(uint64, *ie.CreateURR) error {
	return nil
}

func (Empty) UpdateURR(uint64, *ie.UpdateURR) ([]report.USAReport, error) {
	return nil, nil
}

func (Empty) RemoveURR(uint64, *ie.RemoveURR) ([]report.USAReport, error) {
	return nil, nil
}

func (Empty) CreateBAR(uint64, *ie.CreateBAR) error {
	return nil
}

func (Empty) UpdateBAR(uint64, *ie.UpdateBARSessionModificationRequest) error {
	return nil
}

func (Empty) RemoveBAR(uint64, *ie.RemoveBAR) error {
	return nil
}

func (Empty) QueryURR(uint64, uint32) ([]report.USAReport, error) {
	return nil, nil
}

func (Empty) HandleReport(report.Handler) {
}
