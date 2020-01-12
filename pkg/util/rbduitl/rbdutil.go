package rbdutil

type Labels map[string]string

func (l Labels) WithRainbondLabels() map[string]string {
	rainbondLabels := map[string]string{
		"creator": "Rainbond",
	}
	for k, v := range l {
		rainbondLabels[k] = v
	}
	return rainbondLabels
}

func LabelsForRainbondResource() map[string]string {
	return map[string]string{
		"creator":  "Rainbond",
		"belongTo": "RainbondOperator",
	}
}
