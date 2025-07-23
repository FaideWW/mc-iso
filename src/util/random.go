package util

type RandomSource struct {
	seed int64
}

// RandomSource replicates the Minecraft LegacyRandomSource implementation,
// which in turn is a re-implementation of Java's math.util.Random. This is a
// linear congruential generator (LCG) which is an efficient algorithm for
// producing a sequence of pseudorandom values.
// Here we use the same constants as the Minecraft client in order to produce
// the same variance as the game does when rendering the world.
func NewRandomSource(seed int64) RandomSource {
	return RandomSource{seed}
}

const randomSourceMixingPrime = 25214903917
const randomSourceBitMask = 281474976710655 // (2^48)-1

func (rs *RandomSource) SetSeed(seed int64) {
	newSeed := (seed ^ randomSourceMixingPrime) & randomSourceBitMask
	rs.seed = newSeed
}

func (rs *RandomSource) next(bits int) int32 {
	i := rs.seed
	j := (i*randomSourceMixingPrime + 11) & randomSourceBitMask
	rs.seed = j
	return int32(j >> (48 - bits))
}

func (rs *RandomSource) NextLong() int64 {
	i := rs.next(32)
	j := rs.next(32)
	k := int64(i) << 32
	return k + int64(j)
}

// Mth.getSeed in the Minecraft client source takes a position in world-coordinates and returns a pseudorandom seed value for that block, which is used to variate block models and textures when applicable.
func GetSeed(x, y, z int) int64 {
	i := int64(x)*3129871 ^ int64(z)*116129781 ^ int64(y)
	i = i*i*42317861 + i*11
	return i >> 16
}
