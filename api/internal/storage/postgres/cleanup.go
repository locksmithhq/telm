package postgres

import (
	"context"
	"fmt"
)

// LogRetentionDays é o número de dias de logs mantidos no banco.
const LogRetentionDays = 8

// CleanupLogs remove registros de logs mais antigos que LogRetentionDays e executa
// VACUUM FULL na tabela para devolver espaço físico ao sistema operacional.
//
// O DELETE marca as linhas como mortas; o VACUUM FULL reescreve a tabela
// compactamente e libera os blocos para o OS — diferente do autovacuum padrão
// que só retorna espaço ao free-space map interno do Postgres.
//
// VACUUM FULL adquire lock exclusivo na tabela durante a execução.
// Deve ser chamado em horário de baixo tráfego (ex: 03:00).
func (c *Client) CleanupLogs(ctx context.Context) (deleted int64, err error) {
	res, err := c.db.ExecContext(ctx,
		"DELETE FROM logs WHERE timestamp < NOW() - ($1 * INTERVAL '1 day')",
		LogRetentionDays,
	)
	if err != nil {
		return 0, fmt.Errorf("delete logs: %w", err)
	}
	deleted, _ = res.RowsAffected()

	// VACUUM FULL não pode rodar dentro de uma transação.
	// ExecContext no pool (sem tx ativa) é executado fora de transação — correto.
	if _, err = c.db.ExecContext(ctx, "VACUUM (FULL, ANALYZE) logs"); err != nil {
		return deleted, fmt.Errorf("vacuum logs: %w", err)
	}

	return deleted, nil
}
