package domain

type Data struct {
	Raw []byte
}

func NewData(raw []byte) Data {
	return Data{Raw: raw}
}
