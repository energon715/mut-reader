package readers

import "errors"

// Bound the returned model, including repeated references. ZIP/input limits
// alone do not bound the size of bridge JSON or rendered images.
const MaxDocumentBytes = 32 << 20

type contentBudget struct {
	bytes, blocks, tokens int
	err                   error
}

func (b *contentBudget) take(n int) bool {
	if b.err != nil {
		return false
	}
	if n < 0 || n > MaxDocumentBytes-b.bytes {
		b.err = errors.New("decoded book exceeds 32 MiB model limit")
		return false
	}
	b.bytes += n
	return true
}
func (b *contentBudget) block(n int) bool {
	if b.blocks >= MaxBlocks {
		b.err = errors.New("too many book blocks")
		return false
	}
	if !b.take(n + 64) {
		return false
	}
	b.blocks++
	return true
}

func (b *contentBudget) token() bool {
	if b.tokens >= 500000 {
		b.err = errors.New("book token limit exceeded")
		return false
	}
	b.tokens++
	return true
}
