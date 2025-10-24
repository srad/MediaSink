package responses

type PresetDescription struct {
	Preset      string `json:"preset" extensions:"!x-nullable"`
	Label       string `json:"label" extensions:"!x-nullable"`
	Description string `json:"description" extensions:"!x-nullable"`
	EncodeSpeed string `json:"encodeSpeed" extensions:"!x-nullable"`
}

type CRFDescription struct {
	Value       uint    `json:"value" extensions:"!x-nullable"`
	Label       string  `json:"label" extensions:"!x-nullable"`
	Description string  `json:"description" extensions:"!x-nullable"`
	Quality     string  `json:"quality" extensions:"!x-nullable"`
	ApproxRatio float64 `json:"approxRatio" extensions:"!x-nullable"`
}

type ResolutionDescription struct {
	Resolution  string `json:"resolution" extensions:"!x-nullable"`
	Dimensions  string `json:"dimensions" extensions:"!x-nullable"`
	Description string `json:"description" extensions:"!x-nullable"`
	UseCase     string `json:"useCase" extensions:"!x-nullable"`
}

type FilterDescription[T any] struct {
	Name        string `json:"name" extensions:"!x-nullable"`
	Description string `json:"description" extensions:"!x-nullable"`
	Recommended T      `json:"recommended" extensions:"!x-nullable"`
	Range       string `json:"range" extensions:"!x-nullable"`
	MinValue    T      `json:"minValue" extensions:"!x-nullable"`
	MaxValue    T      `json:"maxValue" extensions:"!x-nullable"`
}

type FilterDescriptions struct {
	DenoiseStrength FilterDescription[float64] `json:"denoiseStrength" extensions:"!x-nullable"`
	SharpenStrength FilterDescription[float64] `json:"sharpenStrength" extensions:"!x-nullable"`
	ApplyNormalize  FilterDescription[bool]    `json:"applyNormalize" extensions:"!x-nullable"`
}

type EnhancementDescriptions struct {
	Presets     [7]PresetDescription  `json:"presets" extensions:"!x-nullable"`
	CRFValues   [5]CRFDescription     `json:"crfValues" extensions:"!x-nullable"`
	Resolutions [4]ResolutionDescription `json:"resolutions" extensions:"!x-nullable"`
	Filters     FilterDescriptions    `json:"filters" extensions:"!x-nullable"`
}
