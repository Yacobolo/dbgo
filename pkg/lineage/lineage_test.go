package lineage

import (
	"testing"
)

// =============================================================================
// Test Helpers
// =============================================================================

func findColumn(cols []*ColumnLineage, name string) *ColumnLineage {
	for _, c := range cols {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// colSpec defines expected column properties for table-driven tests
type colSpec struct {
	name      string
	transform TransformType
	function  string // expected function name (empty = don't check)
	srcCount  *int   // expected source count (nil = don't check)
	srcTable  string // expected first source table (empty = don't check)
}

// srcN is a helper to create a pointer to an int for srcCount
func srcN(n int) *int { return &n }

// testCase defines a single lineage test case
type testCase struct {
	name    string
	sql     string
	schema  Schema
	sources []string  // expected source tables
	cols    []colSpec // expected columns
}

// runLineageTests executes table-driven lineage tests
func runLineageTests(t *testing.T, tests []testCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lineage, err := ExtractLineage(tt.sql, tt.schema)
			if err != nil {
				t.Fatalf("ExtractLineage failed: %v", err)
			}

			// Check sources
			for _, src := range tt.sources {
				if !contains(lineage.Sources, src) {
					t.Errorf("missing source %q, got %v", src, lineage.Sources)
				}
			}

			// Check column count if cols specified
			if tt.cols != nil && len(lineage.Columns) != len(tt.cols) {
				t.Errorf("expected %d columns, got %d", len(tt.cols), len(lineage.Columns))
			}

			// Check each column spec
			for _, spec := range tt.cols {
				col := findColumn(lineage.Columns, spec.name)
				if col == nil {
					t.Errorf("missing column %q", spec.name)
					continue
				}
				if col.Transform != spec.transform {
					t.Errorf("column %q: expected transform %v, got %v", spec.name, spec.transform, col.Transform)
				}
				if spec.function != "" && col.Function != spec.function {
					t.Errorf("column %q: expected function %q, got %q", spec.name, spec.function, col.Function)
				}
				if spec.srcCount != nil && len(col.Sources) != *spec.srcCount {
					t.Errorf("column %q: expected %d sources, got %d", spec.name, *spec.srcCount, len(col.Sources))
				}
				if spec.srcTable != "" && len(col.Sources) > 0 && col.Sources[0].Table != spec.srcTable {
					t.Errorf("column %q: expected source table %q, got %q", spec.name, spec.srcTable, col.Sources[0].Table)
				}
			}
		})
	}
}

// =============================================================================
// Table-Driven Tests
// =============================================================================

func TestExtractLineage_BasicSelects(t *testing.T) {
	runLineageTests(t, []testCase{
		{
			name:    "simple columns",
			sql:     `SELECT id, name, email FROM users`,
			sources: []string{"users"},
			cols: []colSpec{
				{name: "id", transform: TransformDirect},
				{name: "name", transform: TransformDirect},
				{name: "email", transform: TransformDirect},
			},
		},
		{
			name:    "qualified columns",
			sql:     `SELECT u.id, u.name FROM users u`,
			sources: []string{"users"},
			cols: []colSpec{
				{name: "id", transform: TransformDirect, srcTable: "users"},
				{name: "name", transform: TransformDirect, srcTable: "users"},
			},
		},
		{
			name:    "schema qualified table",
			sql:     `SELECT id, name FROM public.users`,
			sources: []string{"public.users"},
			cols: []colSpec{
				{name: "id", transform: TransformDirect},
				{name: "name", transform: TransformDirect},
			},
		},
		{
			name:    "catalog.schema.table",
			sql:     `SELECT id FROM mydb.myschema.users`,
			sources: []string{"mydb.myschema.users"},
			cols:    []colSpec{{name: "id", transform: TransformDirect}},
		},
	})
}

func TestExtractLineage_Expressions(t *testing.T) {
	runLineageTests(t, []testCase{
		{
			name:    "binary expression",
			sql:     `SELECT price * quantity AS total FROM order_items`,
			sources: []string{"order_items"},
			cols:    []colSpec{{name: "total", transform: TransformExpression, srcCount: srcN(2)}},
		},
		{
			name:    "scalar function UPPER",
			sql:     `SELECT UPPER(name) AS upper_name FROM users`,
			sources: []string{"users"},
			cols:    []colSpec{{name: "upper_name", transform: TransformDirect}},
		},
		{
			name:    "COALESCE multiple cols",
			sql:     `SELECT COALESCE(nickname, name) AS display_name FROM users`,
			sources: []string{"users"},
			cols:    []colSpec{{name: "display_name", transform: TransformExpression, srcCount: srcN(2)}},
		},
		{
			name:    "CAST expression",
			sql:     `SELECT CAST(id AS VARCHAR) AS id_str FROM users`,
			sources: []string{"users"},
			cols:    []colSpec{{name: "id_str", transform: TransformExpression}},
		},
		{
			name:    "CASE expression",
			sql:     `SELECT id, CASE WHEN status = 'active' THEN 'Active' ELSE 'Unknown' END AS status_label FROM users`,
			sources: []string{"users"},
			cols: []colSpec{
				{name: "id", transform: TransformDirect},
				{name: "status_label", transform: TransformExpression},
			},
		},
		{
			name:    "literal values",
			sql:     `SELECT id, 'constant' AS label, 42 AS magic_number FROM users`,
			sources: []string{"users"},
			cols: []colSpec{
				{name: "id", transform: TransformDirect},
				{name: "label", transform: TransformExpression, srcCount: srcN(0)},
				{name: "magic_number", transform: TransformExpression, srcCount: srcN(0)},
			},
		},
		{
			name:    "generator functions",
			sql:     `SELECT id, NOW() AS current_time, RANDOM() AS rand_val FROM users`,
			sources: []string{"users"},
			cols: []colSpec{
				{name: "id", transform: TransformDirect},
				{name: "current_time", transform: TransformExpression, srcCount: srcN(0)},
				{name: "rand_val", transform: TransformExpression, srcCount: srcN(0)},
			},
		},
	})
}

func TestExtractLineage_Aggregates(t *testing.T) {
	runLineageTests(t, []testCase{
		{
			name:    "COUNT(*)",
			sql:     `SELECT customer_id, COUNT(*) AS order_count FROM orders GROUP BY customer_id`,
			sources: []string{"orders"},
			cols: []colSpec{
				{name: "customer_id", transform: TransformDirect},
				{name: "order_count", transform: TransformExpression, function: "count"},
			},
		},
		{
			name:    "SUM",
			sql:     `SELECT customer_id, SUM(amount) AS total_amount FROM orders GROUP BY customer_id`,
			sources: []string{"orders"},
			cols: []colSpec{
				{name: "customer_id", transform: TransformDirect},
				{name: "total_amount", transform: TransformExpression, function: "sum"},
			},
		},
		{
			name:    "AVG",
			sql:     `SELECT product_id, AVG(price) AS avg_price FROM products GROUP BY product_id`,
			sources: []string{"products"},
			cols: []colSpec{
				{name: "product_id", transform: TransformDirect},
				{name: "avg_price", transform: TransformExpression, function: "avg"},
			},
		},
	})
}

func TestExtractLineage_WindowFunctions(t *testing.T) {
	runLineageTests(t, []testCase{
		{
			name: "SUM OVER",
			sql: `SELECT id, amount, SUM(amount) OVER (PARTITION BY customer_id ORDER BY created_at) AS running_total
			      FROM orders`,
			sources: []string{"orders"},
			cols: []colSpec{
				{name: "id", transform: TransformDirect},
				{name: "amount", transform: TransformDirect},
				{name: "running_total", transform: TransformExpression},
			},
		},
		{
			name:    "ROW_NUMBER",
			sql:     `SELECT id, ROW_NUMBER() OVER (ORDER BY created_at) AS row_num FROM users`,
			sources: []string{"users"},
			cols: []colSpec{
				{name: "id", transform: TransformDirect},
				{name: "row_num", transform: TransformExpression},
			},
		},
	})
}

func TestExtractLineage_CTEs(t *testing.T) {
	runLineageTests(t, []testCase{
		{
			name: "simple CTE",
			sql: `WITH active_users AS (
				SELECT id, name FROM users WHERE status = 'active'
			)
			SELECT id, name FROM active_users`,
			sources: []string{"users"},
			cols: []colSpec{
				{name: "id", transform: TransformDirect},
				{name: "name", transform: TransformDirect},
			},
		},
		{
			name: "multiple CTEs",
			sql: `WITH 
				customers AS (SELECT id, name FROM users WHERE type = 'customer'),
				orders_summary AS (SELECT customer_id, SUM(amount) AS total FROM orders GROUP BY customer_id)
			SELECT c.name, o.total
			FROM customers c
			JOIN orders_summary o ON c.id = o.customer_id`,
			sources: []string{"users", "orders"},
			cols: []colSpec{
				{name: "name", transform: TransformDirect},
				{name: "total", transform: TransformDirect}, // Direct from CTE column
			},
		},
		{
			name: "CTE with aggregation",
			sql: `WITH daily_totals AS (
				SELECT DATE(created_at) AS day, SUM(amount) AS total
				FROM transactions
				GROUP BY DATE(created_at)
			)
			SELECT day, total FROM daily_totals`,
			sources: []string{"transactions"},
			cols: []colSpec{
				{name: "day", transform: TransformDirect},   // Direct from CTE column
				{name: "total", transform: TransformDirect}, // Direct from CTE column
			},
		},
	})
}

func TestExtractLineage_Joins(t *testing.T) {
	runLineageTests(t, []testCase{
		{
			name: "inner join",
			sql: `SELECT u.name, o.amount
			      FROM users u
			      INNER JOIN orders o ON u.id = o.user_id`,
			sources: []string{"users", "orders"},
			cols: []colSpec{
				{name: "name", transform: TransformDirect, srcTable: "users"},
				{name: "amount", transform: TransformDirect, srcTable: "orders"},
			},
		},
		{
			name: "left join with COALESCE",
			sql: `SELECT c.name, COALESCE(SUM(o.amount), 0) AS total_orders
			      FROM customers c
			      LEFT JOIN orders o ON c.id = o.customer_id
			      GROUP BY c.name`,
			sources: []string{"customers", "orders"},
			cols: []colSpec{
				{name: "name", transform: TransformDirect},
				{name: "total_orders", transform: TransformDirect}, // COALESCE returns direct when single source
			},
		},
		{
			name: "multiple joins",
			sql: `SELECT c.name AS customer_name, p.name AS product_name, oi.quantity
			      FROM customers c
			      JOIN orders o ON c.id = o.customer_id
			      JOIN order_items oi ON o.id = oi.order_id
			      JOIN products p ON oi.product_id = p.id`,
			sources: []string{"customers", "orders", "order_items", "products"},
			cols: []colSpec{
				{name: "customer_name", transform: TransformDirect},
				{name: "product_name", transform: TransformDirect},
				{name: "quantity", transform: TransformDirect},
			},
		},
	})
}

func TestExtractLineage_SetOperations(t *testing.T) {
	runLineageTests(t, []testCase{
		{
			name: "UNION",
			sql: `SELECT id, name FROM customers
			      UNION
			      SELECT id, name FROM suppliers`,
			sources: []string{"customers", "suppliers"},
			cols: []colSpec{
				{name: "id", transform: TransformExpression},
				{name: "name", transform: TransformExpression},
			},
		},
		{
			name: "UNION ALL",
			sql: `SELECT id, email FROM users
			      UNION ALL
			      SELECT id, email FROM archived_users`,
			sources: []string{"users", "archived_users"},
			cols: []colSpec{
				{name: "id", transform: TransformExpression},
				{name: "email", transform: TransformExpression},
			},
		},
		{
			name: "EXCEPT",
			sql: `SELECT id FROM all_users
			      EXCEPT
			      SELECT id FROM blocked_users`,
			sources: []string{"all_users", "blocked_users"},
			cols:    []colSpec{{name: "id", transform: TransformExpression}},
		},
	})
}

func TestExtractLineage_StarExpansion(t *testing.T) {
	runLineageTests(t, []testCase{
		{
			name:    "star without schema",
			sql:     `SELECT * FROM users`,
			sources: []string{"users"},
			cols:    []colSpec{{name: "*", transform: TransformDirect}},
		},
		{
			name:    "star with schema",
			sql:     `SELECT * FROM users`,
			schema:  Schema{"users": {"id", "name", "email", "created_at"}},
			sources: []string{"users"},
			cols: []colSpec{
				{name: "id", transform: TransformDirect},
				{name: "name", transform: TransformDirect},
				{name: "email", transform: TransformDirect},
				{name: "created_at", transform: TransformDirect},
			},
		},
		{
			name: "table.star with schema",
			sql:  `SELECT u.*, o.amount FROM users u JOIN orders o ON u.id = o.user_id`,
			schema: Schema{
				"users":  {"id", "name"},
				"orders": {"id", "user_id", "amount"},
			},
			sources: []string{"users", "orders"},
			cols: []colSpec{
				{name: "id", transform: TransformDirect},
				{name: "name", transform: TransformDirect},
				{name: "amount", transform: TransformDirect},
			},
		},
	})
}

func TestExtractLineage_DerivedTables(t *testing.T) {
	runLineageTests(t, []testCase{
		{
			name: "simple derived table",
			sql: `SELECT sub.id, sub.total
			      FROM (
			          SELECT customer_id AS id, SUM(amount) AS total
			          FROM orders
			          GROUP BY customer_id
			      ) sub`,
			sources: []string{"orders"},
			cols: []colSpec{
				{name: "id", transform: TransformDirect},
				{name: "total", transform: TransformDirect}, // Direct from subquery column
			},
		},
		{
			name: "nested derived tables",
			sql: `SELECT final.name, final.order_count
			      FROM (
			          SELECT u.name, counts.order_count
			          FROM users u
			          JOIN (
			              SELECT user_id, COUNT(*) AS order_count
			              FROM orders
			              GROUP BY user_id
			          ) counts ON u.id = counts.user_id
			      ) final`,
			sources: []string{"users", "orders"},
			cols: []colSpec{
				{name: "name", transform: TransformDirect},
				{name: "order_count", transform: TransformDirect}, // Direct from subquery column
			},
		},
	})
}

func TestExtractLineage_ComplexQuery(t *testing.T) {
	sql := `
	WITH monthly_sales AS (
		SELECT 
			DATE_TRUNC('month', o.created_at) AS month,
			p.category_id,
			SUM(oi.quantity * oi.unit_price) AS revenue
		FROM orders o
		JOIN order_items oi ON o.id = oi.order_id
		JOIN products p ON oi.product_id = p.id
		WHERE o.status = 'completed'
		GROUP BY DATE_TRUNC('month', o.created_at), p.category_id
	),
	category_totals AS (
		SELECT 
			c.name AS category_name,
			ms.month,
			ms.revenue,
			SUM(ms.revenue) OVER (PARTITION BY c.id ORDER BY ms.month) AS cumulative_revenue
		FROM monthly_sales ms
		JOIN categories c ON ms.category_id = c.id
	)
	SELECT 
		category_name,
		month,
		revenue,
		cumulative_revenue,
		ROUND(revenue / NULLIF(LAG(revenue) OVER (PARTITION BY category_name ORDER BY month), 0) * 100 - 100, 2) AS growth_pct
	FROM category_totals
	ORDER BY category_name, month`

	runLineageTests(t, []testCase{
		{
			name:    "complex multi-CTE query",
			sql:     sql,
			sources: []string{"orders", "order_items", "products", "categories"},
			cols: []colSpec{
				{name: "category_name", transform: TransformDirect},
				{name: "month", transform: TransformDirect},              // Direct from CTE
				{name: "revenue", transform: TransformDirect},            // Direct from CTE
				{name: "cumulative_revenue", transform: TransformDirect}, // Direct from CTE
				{name: "growth_pct", transform: TransformExpression},
			},
		},
	})
}

// =============================================================================
// Error Cases
// =============================================================================

func TestExtractLineage_Errors(t *testing.T) {
	tests := []struct {
		name string
		sql  string
	}{
		{"invalid SQL", `SELECT FROM WHERE`},
		{"empty SQL", ``},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExtractLineage(tt.sql, nil)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkExtractLineage_Simple(b *testing.B) {
	sql := `SELECT id, name, email FROM users`
	for i := 0; i < b.N; i++ {
		_, _ = ExtractLineage(sql, nil)
	}
}

func BenchmarkExtractLineage_Complex(b *testing.B) {
	sql := `
	WITH cte AS (
		SELECT id, SUM(amount) AS total FROM orders GROUP BY id
	)
	SELECT u.name, c.total
	FROM users u
	JOIN cte c ON u.id = c.id
	WHERE u.status = 'active'`

	for i := 0; i < b.N; i++ {
		_, _ = ExtractLineage(sql, nil)
	}
}
