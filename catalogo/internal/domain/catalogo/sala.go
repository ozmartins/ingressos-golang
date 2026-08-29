package catalogo

// TipoTela é a tecnologia de exibição da sala.
type TipoTela string

const (
	Tela2D   TipoTela = "2D"
	Tela3D   TipoTela = "3D"
	TelaIMAX TipoTela = "IMAX"
	TelaVIP  TipoTela = "VIP"
)

// Sala é um ambiente de exibição dentro de um cinema.
//
// CapacidadeTotal é informativa para quem consulta. Não é usada para validar
// poltronas: o mapa de assentos pertence ao Servico-Estoque.
type Sala struct {
	ID              string
	CinemaID        string
	Numero          int
	TipoTela        TipoTela
	CapacidadeTotal int
}
