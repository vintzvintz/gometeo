package mfmap

type JsonMap struct {
	Name     string
	Idtech   string
	Taxonomy string
	SubZones []geoFeature
	Bbox     Bbox
	Prevs    PrevList
}

func (m *MfMap) BuildJson() (*JsonMap, error) {
	j := JsonMap{
		Name:     m.Data.Info.Name,
		Idtech:   m.Data.Info.IdTechnique,
		Taxonomy: m.Data.Info.Taxonomy,
		SubZones: m.Geography.Features,
		Bbox:     m.Geography.Bbox.Crop(),
		Prevs:    m.Forecasts.ByEcheance(),
	}
	return &j, nil
}
