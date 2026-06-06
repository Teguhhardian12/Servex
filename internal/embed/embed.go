// Package embed provides lightweight embedding via TF-IDF feature hashing.
// Zero model downloads, pure Go, works offline.
// Produces 256-dim vectors suitable for cosine similarity search.
package embed

import (
	"math"
	"strings"
	"unicode"
)

const VecDim = 256

// Vector is a fixed-size feature-hashed TF-IDF vector.
type Vector [VecDim]float32

// Tokenize splits text into lowercase tokens, removing punctuation.
func Tokenize(text string) []string {
	f := func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}
	raw := strings.FieldsFunc(strings.ToLower(text), f)
	out := make([]string, 0, len(raw))
	for _, w := range raw {
		if len(w) > 1 { // skip single chars
			out = append(out, w)
		}
	}
	return out
}

// hash returns a deterministic bucket index for a token.
func hash(token string) uint32 {
	var h uint32 = 2166136261
	for _, c := range token {
		h ^= uint32(c)
		h *= 16777619
	}
	return h
}

// hashSign returns +1 or -1 deterministically for a token (signed hashing trick).
func hashSign(token string) float32 {
	var h uint32 = 5381
	for _, c := range token {
		h = ((h << 5) + h) + uint32(c)
	}
	if h%2 == 0 {
		return 1.0
	}
	return -1.0
}

// Embed computes a feature-hashed TF-IDF vector for a single document.
// idfMap can be nil for pure TF (no corpus statistics).
func Embed(tokens []string, idfMap map[string]float64) Vector {
	var v Vector
	if len(tokens) == 0 {
		return v
	}

	// Count term frequencies
	tf := make(map[string]float64)
	for _, t := range tokens {
		tf[t]++
	}

	// TF-IDF weighted feature hashing
	for token, count := range tf {
		tfVal := count / float64(len(tokens)) // normalized TF
		idfVal := 1.0
		if idfMap != nil {
			if val, ok := idfMap[token]; ok {
				idfVal = val
			}
		}
		weight := float32(tfVal * idfVal)
		idx := hash(token) % VecDim
		sign := hashSign(token)
		v[idx] += weight * sign
	}

	// L2 normalize
	var norm float32
	for _, val := range v {
		norm += val * val
	}
	if norm > 0 {
		norm = float32(math.Sqrt(float64(norm)))
		for i := range v {
			v[i] /= norm
		}
	}

	return v
}

// CosineSimilarity computes cosine similarity between two vectors.
func CosineSimilarity(a, b Vector) float32 {
	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// Serialize encodes a Vector to []byte for SQLite BLOB storage.
func (v Vector) Serialize() []byte {
	b := make([]byte, VecDim*4)
	for i, f := range v {
		bits := math.Float32bits(f)
		b[i*4] = byte(bits)
		b[i*4+1] = byte(bits >> 8)
		b[i*4+2] = byte(bits >> 16)
		b[i*4+3] = byte(bits >> 24)
	}
	return b
}

// DeserializeVector decodes a Vector from a SQLite BLOB.
func DeserializeVector(b []byte) Vector {
	var v Vector
	for i := range v {
		v[i] = math.Float32frombits(
			uint32(b[i*4]) | uint32(b[i*4+1])<<8 | uint32(b[i*4+2])<<16 | uint32(b[i*4+3])<<24,
		)
	}
	return v
}
