package predict

import "math"

// poissonPMF returns P(X = k) for X ~ Poisson(lambda), computed in log-space
// to stay numerically stable for larger k.
func poissonPMF(lambda float64, k int) float64 {
	if lambda <= 0 {
		if k == 0 {
			return 1
		}
		return 0
	}
	logP := -lambda + float64(k)*math.Log(lambda) - logFactorial(k)
	return math.Exp(logP)
}

func logFactorial(k int) float64 {
	lf := 0.0
	for i := 2; i <= k; i++ {
		lf += math.Log(float64(i))
	}
	return lf
}
