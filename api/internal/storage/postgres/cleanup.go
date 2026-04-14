package postgres

import (
	"context"
	"fmt"
)

// Retenção padrão em dias. Ajuste via variável ou rebuild.
const (
	LogRetentionDays    = 8
	TraceRetentionDays  = 7
	MetricRetentionDays = 15
)

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

// CleanupTraces remove spans mais antigos que TraceRetentionDays dias.
func (c *Client) CleanupTraces(ctx context.Context) (deleted int64, err error) {
	res, err := c.db.ExecContext(ctx,
		"DELETE FROM traces WHERE start_time < NOW() - ($1 * INTERVAL '1 day')",
		TraceRetentionDays,
	)
	if err != nil {
		return 0, fmt.Errorf("delete traces: %w", err)
	}
	deleted, _ = res.RowsAffected()

	if _, err = c.db.ExecContext(ctx, "VACUUM (FULL, ANALYZE) traces"); err != nil {
		return deleted, fmt.Errorf("vacuum traces: %w", err)
	}

	return deleted, nil
}

// CleanupMetrics remove pontos de métrica mais antigos que MetricRetentionDays dias.
func (c *Client) CleanupMetrics(ctx context.Context) (deleted int64, err error) {
	res, err := c.db.ExecContext(ctx,
		"DELETE FROM metrics WHERE timestamp < NOW() - ($1 * INTERVAL '1 day')",
		MetricRetentionDays,
	)
	if err != nil {
		return 0, fmt.Errorf("delete metrics: %w", err)
	}
	deleted, _ = res.RowsAffected()

	if _, err = c.db.ExecContext(ctx, "VACUUM (FULL, ANALYZE) metrics"); err != nil {
		return deleted, fmt.Errorf("vacuum metrics: %w", err)
	}

	return deleted, nil
}
