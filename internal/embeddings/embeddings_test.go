package embeddings

import (
	"math"
	"testing"
)

func TestSerializeDeserialize(t *testing.T) {
	original := []float32{1.0, -2.5, 3.14, 0.0, -0.001}
	bytes := SerializeEmbedding(original)

	if len(bytes) != len(original)*4 {
		t.Fatalf("expected %d bytes, got %d", len(original)*4, len(bytes))
	}

	restored := DeserializeEmbedding(bytes)
	if len(restored) != len(original) {
		t.Fatalf("expected %d floats, got %d", len(original), len(restored))
	}

	for i := range original {
		if original[i] != restored[i] {
			t.Errorf("index %d: expected %f, got %f", i, original[i], restored[i])
		}
	}
}

func TestSerializeEmpty(t *testing.T) {
	bytes := SerializeEmbedding(nil)
	if len(bytes) != 0 {
		t.Errorf("expected empty bytes, got %d", len(bytes))
	}

	restored := DeserializeEmbedding(bytes)
	if len(restored) != 0 {
		t.Errorf("expected empty slice, got %d", len(restored))
	}
}

func TestCosineSimilarity_Identical(t *testing.T) {
	v := []float32{1, 2, 3}
	sim := CosineSimilarity(v, v)
	if math.Abs(float64(sim)-1.0) > 0.0001 {
		t.Errorf("expected ~1.0 for identical vectors, got %f", sim)
	}
}

func TestCosineSimilarity_Orthogonal(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}
	sim := CosineSimilarity(a, b)
	if math.Abs(float64(sim)) > 0.0001 {
		t.Errorf("expected ~0.0 for orthogonal vectors, got %f", sim)
	}
}

func TestCosineSimilarity_Opposite(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{-1, -2, -3}
	sim := CosineSimilarity(a, b)
	if math.Abs(float64(sim)+1.0) > 0.0001 {
		t.Errorf("expected ~-1.0 for opposite vectors, got %f", sim)
	}
}

func TestCosineSimilarity_ZeroVector(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{0, 0, 0}
	sim := CosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("expected 0 for zero vector, got %f", sim)
	}
}

func TestCosineSimilarity_DifferentLengths(t *testing.T) {
	a := []float32{1, 2}
	b := []float32{1, 2, 3}
	sim := CosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("expected 0 for mismatched lengths, got %f", sim)
	}
}

func TestCosineSimilarity_Similar(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{1.1, 2.1, 3.1}
	sim := CosineSimilarity(a, b)
	if sim < 0.99 {
		t.Errorf("expected high similarity for similar vectors, got %f", sim)
	}
}
