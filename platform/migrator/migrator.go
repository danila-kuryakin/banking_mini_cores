// Package migrator накатывает SQL-миграции, встроенные в бинарь сервиса.
//
// Источник — любая fs.FS (на практике embed.FS сервиса), приёмник — та же
// DataBaseConfig, из которой поднимается рабочий пул pgx. Версии хранятся в
// таблице schema_migrations, её заводит сам golang-migrate.
package migrator

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/golang-migrate/migrate/v4"
	pgxdb "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib" // регистрирует драйвер "pgx" в database/sql
)

// Up применяет все ещё не накатанные миграции.
// Отсутствие новых миграций ошибкой не считается.
func Up(cfg config.DataBaseConfig, fsys fs.FS, dir string) (err error) {
	m, err := newMigrate(cfg, fsys, dir)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, closeMigrate(m)) }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// Down откатывает все миграции. Таблица schema_migrations при этом остаётся.
func Down(cfg config.DataBaseConfig, fsys fs.FS, dir string) (err error) {
	m, err := newMigrate(cfg, fsys, dir)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, closeMigrate(m)) }()

	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("roll back migrations: %w", err)
	}
	return nil
}

// Steps двигает схему на n шагов: n > 0 — вверх, n < 0 — вниз.
func Steps(cfg config.DataBaseConfig, fsys fs.FS, dir string, n int) (err error) {
	if n == 0 {
		return errors.New("migrations: step must not be zero")
	}

	m, err := newMigrate(cfg, fsys, dir)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, closeMigrate(m)) }()

	if err := m.Steps(n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("move schema by %d steps: %w", n, err)
	}
	return nil
}

// Version возвращает текущую версию схемы. Если не накатано ничего,
// applied будет false, а version и dirty — нулевыми.
//
// dirty == true означает, что предыдущая миграция упала на середине: пока её
// не починят руками и не сбросят через Force, накат работать не будет.
func Version(cfg config.DataBaseConfig, fsys fs.FS, dir string) (version uint, dirty, applied bool, err error) {
	m, err := newMigrate(cfg, fsys, dir)
	if err != nil {
		return 0, false, false, err
	}
	defer func() { err = errors.Join(err, closeMigrate(m)) }()

	version, dirty, err = m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, false, nil
	}
	if err != nil {
		return 0, false, false, fmt.Errorf("read schema version: %w", err)
	}
	return version, dirty, true, nil
}

// Force выставляет версию схемы принудительно и снимает флаг dirty,
// ничего не выполняя. Нужен только чтобы разгрести упавшую миграцию.
func Force(cfg config.DataBaseConfig, fsys fs.FS, dir string, version int) (err error) {
	m, err := newMigrate(cfg, fsys, dir)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, closeMigrate(m)) }()

	if err := m.Force(version); err != nil {
		return fmt.Errorf("force schema version %d: %w", version, err)
	}
	return nil
}

// newMigrate собирает мигратор поверх fsys и конфига БД.
func newMigrate(cfg config.DataBaseConfig, fsys fs.FS, dir string) (*migrate.Migrate, error) {
	src, err := iofs.New(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations from %q: %w", dir, err)
	}

	// GetDSN отдаёт строку вида "host=... port=... ...", а не URL. pgx её
	// принимает, поэтому URL-форма pgx5:// здесь не нужна.
	db, err := sql.Open("pgx", cfg.GetDSN())
	if err != nil {
		return nil, fmt.Errorf("open db connection: %w", err)
	}

	// Пустого конфига достаточно: драйвер сам добирает имя базы и схему
	// запросами и ставит таблицу версий по умолчанию — schema_migrations.
	drv, err := pgxdb.WithInstance(db, &pgxdb.Config{})
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("create migration driver: %w", err),
			db.Close(),
		)
	}

	m, err := migrate.NewWithInstance("iofs", src, "pgx5", drv)
	if err != nil {
		return nil, fmt.Errorf("create migrator: %w", err)
	}
	return m, nil
}

// closeMigrate закрывает источник и соединение: Close возвращает две
// независимые ошибки, и терять любую из них не хочется.
func closeMigrate(m *migrate.Migrate) error {
	srcErr, dbErr := m.Close()
	if srcErr != nil {
		srcErr = fmt.Errorf("close migration source: %w", srcErr)
	}
	if dbErr != nil {
		dbErr = fmt.Errorf("close db connection: %w", dbErr)
	}
	return errors.Join(srcErr, dbErr)
}
