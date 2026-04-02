package dataplane

import (
	"github.com/reogac/pfcp/ie"
	"github.com/reogac/sbi/models"
)

func flowInfo2SDFFilters(direction uint8, flowInfo *models.FlowInformation) (filters []ie.SDFFilter) {
	//TODO: support other type of filter
	//for now just consider as bi-directional flow description
	if fd := flowInfo.FlowDescription; len(fd) > 0 {
		filter := ie.NewSDFFilter(fd, "", "", "", 0)
		filters = append(filters, *filter)
	}
	return
}
