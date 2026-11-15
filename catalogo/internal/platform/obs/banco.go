package obs

import (
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

// Gorm acrescenta um span a cada consulta, filho do span que estiver no
// contexto da chamada (use sempre db.WithContext(ctx)). As variáveis das
// consultas não vão para o span: podem carregar dados pessoais.
//
// Consultas sem span-pai (o polling da caixa de saída, as varreduras) não geram
// trace: ver amostrador.
func Gorm(db *gorm.DB) error {
	return db.Use(tracing.NewPlugin(tracing.WithoutMetrics(), tracing.WithoutQueryVariables()))
}
