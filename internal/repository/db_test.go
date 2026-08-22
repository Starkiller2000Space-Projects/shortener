package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/joho/godotenv"
	"github.com/max-marek-projects/shortener/internal/config/db"
	"github.com/max-marek-projects/shortener/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// create test db in container
func setupTestDB(t interface {
	mock.TestingT
	Skip(...any)
}) (*dbStorage, *db.DBConf, func()) {
	err := godotenv.Load("../../.env")
	if err != nil && !os.IsNotExist(err) {
		require.NoError(t, err)
	}
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping integration test")
	}
	config := db.NewDBConf(dsn, "../../migrations")
	storage, err := NewDBStorage(config)
	require.NoError(t, err)
	cleanup := func() {
		_, err := storage.storage.ExecContext(context.Background(), "TRUNCATE urls")
		if err != nil {
			t.Logf("failed to truncate: %v", err)
		}
	}
	return storage, config, cleanup
}

// test addition to db storage
func TestDBStorage_Add(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	storage := &dbStorage{
		storage: sqlDB,
		config:  &db.DBConf{},
	}

	ctx := context.Background()
	type want struct {
		value string
		err   error
	}
	tests := []struct {
		name       string
		row        Row
		wants      want
		sideEffect error
	}{
		{
			name: "successful insert",
			row:  Row{ID: "id1", OriginalURL: "http://example.com", UserID: "u1"},
			wants: want{
				value: "id1",
			},
		},
		{
			name: "conflict - existing URL",
			row:  Row{ID: "id2", OriginalURL: "http://example.com", UserID: "u2"},
			wants: want{
				value: "existingID",
				err:   &ErrAlreadyExists{ExistingID: "existingID"},
			},
		},
		{
			name:       "db error",
			row:        Row{ID: "id3", OriginalURL: "http://error.com", UserID: "u3"},
			sideEffect: sql.ErrConnDone,
			wants: want{
				value: "",
				err:   sql.ErrConnDone,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			if test.sideEffect != nil {
				mock.ExpectQuery(`INSERT INTO urls`).
					WithArgs(test.row.ID, test.row.OriginalURL, test.row.UserID).
					WillReturnError(test.sideEffect)
			} else {
				mock.ExpectQuery(`INSERT INTO urls`).
					WithArgs(test.row.ID, test.row.OriginalURL, test.row.UserID).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(test.wants.value))
			}

			err := storage.Add(ctx, test.row)
			if test.wants.err != nil {
				assert.Error(t, err)
				switch test.wants.err.(type) {
				case *ErrAlreadyExists:
					var e *ErrAlreadyExists
					assert.ErrorAs(t, err, &e)
					assert.Equal(t, test.wants.err.(*ErrAlreadyExists).ExistingID, e.ExistingID)
				default:
					assert.ErrorIs(t, err, test.wants.err)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func BenchmarkDBStorageAdd(b *testing.B) {
	storage, _, cleanup := setupTestDB(b)
	defer cleanup()

	ctx := context.Background()

	for b.Loop() {
		id := utils.GenerateID(8)
		row := Row{
			ID:          id,
			OriginalURL: "https://example.com/" + id,
			UserID:      "user",
		}
		err := storage.Add(ctx, row)
		if err != nil {
			b.Fatalf("Add failed: %v", err)
		}
	}
}

func TestDBStorage_Get(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		setup     func(mock sqlmock.Sqlmock)
		ctx       context.Context
		want      string
		wantErr   error
		wantErrIs error
	}{
		{
			name: "success",
			id:   "id1",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT original_url, is_deleted FROM urls WHERE id = \$1`).
					WithArgs("id1").
					WillReturnRows(
						sqlmock.NewRows([]string{"original_url", "is_deleted"}).
							AddRow("https://example.com", false),
					)
			},
			ctx:  context.Background(),
			want: "https://example.com",
		},
		{
			name: "not found",
			id:   "missing",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT original_url, is_deleted FROM urls WHERE id = \$1`).
					WithArgs("missing").
					WillReturnError(sql.ErrNoRows)
			},
			ctx:       context.Background(),
			wantErr:   ErrNotFound,
			wantErrIs: ErrNotFound,
		},
		{
			name: "db error",
			id:   "id1",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT original_url, is_deleted FROM urls WHERE id = \$1`).
					WithArgs("id1").
					WillReturnError(sql.ErrConnDone)
			},
			ctx:       context.Background(),
			wantErrIs: sql.ErrConnDone,
		},
		{
			name: "deleted",
			id:   "id1",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT original_url, is_deleted FROM urls WHERE id = \$1`).
					WithArgs("id1").
					WillReturnRows(
						sqlmock.NewRows([]string{"original_url", "is_deleted"}).
							AddRow("https://example.com", true),
					)
			},
			ctx:       context.Background(),
			wantErrIs: ErrGone,
		},
		{
			name:  "context canceled",
			id:    "id1",
			setup: func(mock sqlmock.Sqlmock) {},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			wantErrIs: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer sqlDB.Close()

			storage := &dbStorage{
				storage: sqlDB,
				config:  &db.DBConf{},
			}

			tt.setup(mock)

			got, err := storage.Get(tt.ctx, tt.id)

			if tt.wantErrIs != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErrIs)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func BenchmarkDBStorageGet(b *testing.B) {
	storage, _, cleanup := setupTestDB(b)
	defer cleanup()

	ctx := context.Background()
	id := "fixedID"
	row := Row{ID: id, OriginalURL: "https://example.com/fixed", UserID: "user"}
	err := storage.Add(ctx, row)
	if err != nil {
		b.Fatalf("failed to add initial row: %v", err)
	}

	for b.Loop() {
		_, err := storage.Get(ctx, id)
		if err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}
}

// test ping for file storage
func TestDBStorage_Ping(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
	}{{name: "success", err: nil}, {name: "ping error", err: sql.ErrConnDone}} {
		t.Run(test.name, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err)
			defer sqlDB.Close()

			storage := &dbStorage{
				storage: sqlDB,
				config:  &db.DBConf{},
			}
			ctx := context.Background()

			mock.ExpectPing().WillReturnError(test.err)
			err = storage.Ping(ctx)
			if test.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.ErrorIs(t, err, test.err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// test batch addition to file storage
func TestDBStorage_AddBatch(t *testing.T) {
	tests := []struct {
		name    string
		rows    []Row
		mockErr error
		wantErr error
	}{
		{
			name: "successful batch",
			rows: []Row{
				{ID: "id1", OriginalURL: "http://example1.com", UserID: "u1"},
				{ID: "id2", OriginalURL: "http://example2.com", UserID: "u2"},
			},
			wantErr: nil,
		},
		{
			name:    "empty batch",
			rows:    []Row{},
			wantErr: nil,
		},
		{
			name: "db error",
			rows: []Row{
				{ID: "id3", OriginalURL: "http://error.com", UserID: "u3"},
			},
			mockErr: sql.ErrConnDone,
			wantErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer sqlDB.Close()

			storage := &dbStorage{
				storage: sqlDB,
				config:  &db.DBConf{},
			}
			ctx := context.Background()

			if tt.mockErr != nil {
				mock.ExpectExec(`INSERT INTO urls`).
					WillReturnError(tt.mockErr)
			} else if len(tt.rows) > 0 {
				args := make([]driver.Value, 0, len(tt.rows)*3)
				for _, row := range tt.rows {
					args = append(args, row.ID, row.OriginalURL, row.UserID)
				}
				mock.ExpectExec(`INSERT INTO urls`).
					WithArgs(args...).
					WillReturnResult(sqlmock.NewResult(1, int64(len(tt.rows))))
			}

			err = storage.AddBatch(ctx, tt.rows)
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDBStorage_buildBatchInsertQuery(t *testing.T) {
	tests := []struct {
		name      string
		items     []Row
		wantQuery string
		wantArgs  []any
	}{
		{
			name:      "empty",
			items:     nil,
			wantQuery: `INSERT INTO urls(id, original_url, user_id, is_deleted) VALUES `,
			wantArgs:  []any{},
		},
		{
			name: "single row",
			items: []Row{
				{
					ID:          "id1",
					OriginalURL: "http://example.com",
					UserID:      "u1",
				},
			},
			wantQuery: `INSERT INTO urls(id, original_url, user_id, is_deleted) VALUES ($1, $2, $3, false)`,
			wantArgs: []any{
				"id1",
				"http://example.com",
				"u1",
			},
		},
		{
			name: "multiple rows",
			items: []Row{
				{
					ID:          "id1",
					OriginalURL: "http://example1.com",
					UserID:      "u1",
				},
				{
					ID:          "id2",
					OriginalURL: "http://example2.com",
					UserID:      "u2",
				},
			},
			wantQuery: `INSERT INTO urls(id, original_url, user_id, is_deleted) VALUES ($1, $2, $3, false), ($4, $5, $6, false)`,
			wantArgs: []any{
				"id1",
				"http://example1.com",
				"u1",
				"id2",
				"http://example2.com",
				"u2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &dbStorage{}

			gotQuery, gotArgs, err := storage.buildBatchInsertQuery(tt.items)

			require.NoError(t, err)
			assert.Equal(t, tt.wantQuery, gotQuery)
			assert.Equal(t, tt.wantArgs, gotArgs)
		})
	}
}

func BenchmarkDBStorageAddBatch(b *testing.B) {
	storage, _, cleanup := setupTestDB(b)
	defer cleanup()

	ctx := context.Background()

	for i := 0; b.Loop(); i++ {
		items := make([]Row, 10)
		for j := range 10 {
			items[j] = Row{
				ID:          utils.GenerateID(8),
				OriginalURL: "https://example.com/" + fmt.Sprint(i) + "/" + fmt.Sprint(j),
				UserID:      "user",
			}
		}
		err := storage.AddBatch(ctx, items)
		if err != nil {
			b.Fatalf("AddBatch failed: %v", err)
		}
	}
}

// test get user urls
func TestDBStorage_GetUserUrls(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		mockRows *sqlmock.Rows
		mockErr  error
		want     []UserURL
		wantErr  error
	}{
		{
			name:   "found",
			userID: "u1",
			mockRows: sqlmock.NewRows([]string{"id", "original_url"}).
				AddRow("id1", "http://example1.com").
				AddRow("id2", "http://example2.com"),
			want: []UserURL{
				{ShortURL: "id1", OriginalURL: "http://example1.com"},
				{ShortURL: "id2", OriginalURL: "http://example2.com"},
			},
			wantErr: nil,
		},
		{
			name:     "no rows",
			userID:   "u2",
			mockRows: sqlmock.NewRows([]string{"id", "original_url"}),
			want:     []UserURL(nil),
			wantErr:  nil,
		},
		{
			name:    "db error",
			userID:  "u3",
			mockErr: sql.ErrConnDone,
			wantErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer sqlDB.Close()

			storage := &dbStorage{
				storage: sqlDB,
				config:  &db.DBConf{},
			}
			ctx := context.Background()

			if tt.mockErr != nil {
				mock.ExpectQuery(`SELECT id, original_url FROM urls WHERE user_id = \$1 AND is_deleted = false`).
					WithArgs(tt.userID).
					WillReturnError(tt.mockErr)
			} else {
				mock.ExpectQuery(`SELECT id, original_url FROM urls WHERE user_id = \$1 AND is_deleted = false`).
					WithArgs(tt.userID).
					WillReturnRows(tt.mockRows)
			}

			got, err := storage.GetUserURLs(ctx, tt.userID)
			if tt.wantErr == nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			} else {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func BenchmarkDBStorageGetUserURLs(b *testing.B) {
	storage, _, cleanup := setupTestDB(b)
	defer cleanup()

	ctx := context.Background()
	userID := "benchUser"
	for i := 0; i < 100; i++ {
		row := Row{
			ID:          utils.GenerateID(8),
			OriginalURL: "https://example.com/" + fmt.Sprint(i),
			UserID:      userID,
		}
		err := storage.Add(ctx, row)
		if err != nil {
			b.Fatalf("failed to add initial data: %v", err)
		}
	}

	for b.Loop() {
		_, err := storage.GetUserURLs(ctx, userID)
		if err != nil {
			b.Fatalf("GetUserURLs failed: %v", err)
		}
	}
}

// test delete batch from storage
func TestDBStorage_DeleteBatch(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		shortIDs []string
		mockErr  error
		wantErr  error
	}{
		{
			name:     "successful delete",
			userID:   "u1",
			shortIDs: []string{"id1", "id2"},
			wantErr:  nil,
		},
		{
			name:     "empty ids",
			userID:   "u2",
			shortIDs: []string{},
			wantErr:  nil,
		},
		{
			name:     "db error",
			userID:   "u3",
			shortIDs: []string{"id3"},
			mockErr:  sql.ErrConnDone,
			wantErr:  sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer sqlDB.Close()

			storage := &dbStorage{
				storage: sqlDB,
				config:  &db.DBConf{},
			}
			ctx := context.Background()

			if tt.mockErr != nil {
				mock.ExpectExec(`UPDATE urls SET is_deleted = true WHERE user_id = \$1 AND id = ANY\(\$2\)`).
					WithArgs(tt.userID, sqlmock.AnyArg()).
					WillReturnError(tt.mockErr)
			} else if len(tt.shortIDs) > 0 {
				mock.ExpectExec(`UPDATE urls SET is_deleted = true WHERE user_id = \$1 AND id = ANY\(\$2\)`).
					WithArgs(tt.userID, sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, int64(len(tt.shortIDs))))
			}

			err = storage.DeleteBatch(ctx, tt.userID, tt.shortIDs)
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func BenchmarkDBStorageDeleteBatch(b *testing.B) {
	storage, _, cleanup := setupTestDB(b)
	defer cleanup()

	ctx := context.Background()
	userID := "delUser"
	ids := make([]string, 10)
	for i := 0; i < 10; i++ {
		id := utils.GenerateID(8)
		ids[i] = id
		row := Row{ID: id, OriginalURL: "https://example.com/" + id, UserID: userID}
		err := storage.Add(ctx, row)
		if err != nil {
			b.Fatalf("failed to add initial data: %v", err)
		}
	}

	for b.Loop() {
		err := storage.DeleteBatch(ctx, userID, ids)
		if err != nil {
			b.Fatalf("DeleteBatch failed: %v", err)
		}
		for _, id := range ids {
			row := Row{ID: id, OriginalURL: "https://example.com/" + id, UserID: userID}
			err := storage.Add(ctx, row)
			if err != nil {
				b.Fatalf("failed to re-add after delete: %v", err)
			}
		}
	}
}

func TestDBStorage_GetStats(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer sqlDB.Close()

	storage := &dbStorage{
		storage: sqlDB,
		config:  &db.DBConf{},
	}

	mock.ExpectBegin()

	rows := sqlmock.NewRows([]string{"count"}).AddRow(10)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM urls WHERE is_deleted = false`).WillReturnRows(rows)

	rowsUsers := sqlmock.NewRows([]string{"count"}).AddRow(3)
	mock.ExpectQuery(`SELECT COUNT\(DISTINCT user_id\) FROM urls WHERE is_deleted = false`).WillReturnRows(rowsUsers)

	mock.ExpectCommit()

	stats, err := storage.GetStats(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 10, stats.URLs)
	assert.Equal(t, 3, stats.Users)
	assert.NoError(t, mock.ExpectationsWereMet())
}
