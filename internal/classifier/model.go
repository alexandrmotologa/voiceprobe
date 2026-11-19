package classifier

import (
	"math"
	"sync"
)

// DenseLayer represents a fully-connected neural network layer.
type DenseLayer struct {
	InFeatures  int
	OutFeatures int
	Weights     [][]float32
	Biases      []float32
}

// ForwardComputes output = ReLU(Weights * input + Biases) into pre-allocated dst.
func (l *DenseLayer) Forward(in []float32, dst []float32, relu bool) {
	for i := 0; i < l.OutFeatures; i++ {
		var sum float32
		wRow := l.Weights[i]
		for j := 0; j < l.InFeatures; j++ {
			sum += wRow[j] * in[j]
		}
		sum += l.Biases[i]

		if relu {
			if sum < 0 {
				sum = 0
			}
		}
		dst[i] = sum
	}
}

// NeuralClassifier defines a feedforward acoustic fraud classification model with reusable scratch buffers.
type NeuralClassifier struct {
	mu     sync.Mutex
	layer1 DenseLayer // 48 -> 32
	layer2 DenseLayer // 32 -> 16
	layer3 DenseLayer // 16 -> 1
	h1Buf  []float32
	h2Buf  []float32
	outBuf []float32
}

// NewNeuralClassifier builds the network with provided pre-calibrated weights and pre-allocated evaluation buffers.
func NewNeuralClassifier(w1 [][]float32, b1 []float32, w2 [][]float32, b2 []float32, w3 [][]float32, b3 []float32) *NeuralClassifier {
	return &NeuralClassifier{
		layer1: DenseLayer{InFeatures: 48, OutFeatures: 32, Weights: w1, Biases: b1},
		layer2: DenseLayer{InFeatures: 32, OutFeatures: 16, Weights: w2, Biases: b2},
		layer3: DenseLayer{InFeatures: 16, OutFeatures: 1, Weights: w3, Biases: b3},
		h1Buf:  make([]float32, 32),
		h2Buf:  make([]float32, 16),
		outBuf: make([]float32, 1),
	}
}

// Predict evaluates the acoustic feature vector and outputs a synthetic probability score in [0.0, 1.0].
func (nc *NeuralClassifier) Predict(features []float32) float64 {
	if len(features) < 48 {
		return 0.0
	}

	nc.mu.Lock()
	defer nc.mu.Unlock()

	nc.layer1.Forward(features, nc.h1Buf, true)
	nc.layer2.Forward(nc.h1Buf, nc.h2Buf, true)
	nc.layer3.Forward(nc.h2Buf, nc.outBuf, false)

	z := float64(nc.outBuf[0])
	prob := 1.0 / (1.0 + math.Exp(-z))
	if math.IsNaN(prob) || math.IsInf(prob, 0) {
		return 0.0
	}
	return prob
}
