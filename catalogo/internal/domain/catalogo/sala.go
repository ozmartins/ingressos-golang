package catalogo

type TipoTela string

const (
	Tela2D   TipoTela = "2D"
	Tela3D   TipoTela = "3D"
	TelaIMAX TipoTela = "IMAX"
	TelaVIP  TipoTela = "VIP"
)

type Sala struct {
	ID              string
	CinemaID        string
	Numero          int
	TipoTela        TipoTela
	CapacidadeTotal int
}
