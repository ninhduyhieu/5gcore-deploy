package nasbuilder

import (
	"github.com/reogac/nas"
	"github.com/reogac/utils"
)

type GmmComposer[T nas.GmmMessage] struct {
	msg   T
	ctx   *nas.NasContext
	isGpp bool
}

func NewGmmComposer[T nas.GmmMessage](ctx *nas.NasContext, isGpp bool, msg T) *GmmComposer[T] {
	return &GmmComposer[T]{
		msg:   msg,
		ctx:   ctx,
		isGpp: isGpp,
	}
}

type NasMmBuilder[T nas.GmmMessage] func(T) error

func (c *GmmComposer[T]) Build(builders ...NasMmBuilder[T]) ([]byte, error) {
	for _, b := range builders {
		if err := b(c.msg); err != nil {
			return nil, err
		}
	}
	if pdu, err := nas.EncodeMm(c.ctx, c.msg, c.isGpp); err != nil {
		return nil, utils.WrapError("Encode N1Mm", err)
	} else {
		return pdu, nil
	}
}
