package stream

import (
	"context"

	ds "github.com/m-rots/bernard/datastore"
	"github.com/m-rots/bernard/datastore/sqlite"

	// database driver
	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	*sqlite.Datastore
}

func NewStore(path string) (store Store, err error) {
	datastore, err := sqlite.New(path)
	if err != nil {
		return
	}

	store = Store{datastore}
	if err = store.createFolderParentIndex(); err != nil {
		return
	}

	if err = store.createFileParentIndex(); err != nil {
		return
	}

	return store, nil
}

const sqlFileParentIndex = `
CREATE INDEX IF NOT EXISTS file_parent ON file(parent)
`

func (s Store) createFileParentIndex() error {
	_, err := s.DB.Exec(sqlFileParentIndex)
	return err
}

const sqlFolderParentIndex = `
CREATE INDEX IF NOT EXISTS folder_parent ON folder(parent)
`

func (s Store) createFolderParentIndex() error {
	_, err := s.DB.Exec(sqlFolderParentIndex)
	return err
}

const sqlGetFile = `
SELECT id, name, size, md5 FROM file WHERE file.id = ? AND NOT file.trashed
`

func (s Store) GetFile(ctx context.Context, id string) (ds.File, error) {
	f := ds.File{}

	row := s.DB.QueryRowContext(ctx, sqlGetFile, id)
	err := row.Scan(&f.ID, &f.Name, &f.Size, &f.MD5)
	return f, err
}

const sqlRecursiveFiles = `
WITH cte AS (
	SELECT id FROM folder WHERE parent = ? AND NOT trashed
	UNION
	SELECT folder.id FROM folder, cte WHERE folder.parent = cte.id AND NOT trashed
)
SELECT id, name, size, md5 FROM file WHERE file.parent IN cte AND NOT file.trashed
`

// RecursiveFiles retrieves all files recursively from the datastore.
// Should be used to get all children of a TV show as well as all films.
func (s Store) RecursiveFiles(ctx context.Context, id string) (files []ds.File, err error) {
	rows, err := s.DB.QueryContext(ctx, sqlRecursiveFiles, id)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		f := ds.File{}
		err := rows.Scan(&f.ID, &f.Name, &f.Size, &f.MD5)
		if err != nil {
			return nil, err
		}

		files = append(files, f)
	}

	return files, rows.Err()
}

const sqlRecursiveFolders = `
WITH cte AS (
	SELECT *, 1 AS depth FROM folder WHERE parent = $1 AND NOT trashed
	UNION
	SELECT folder.*, (cte.depth + 1) AS depth FROM folder, cte
	WHERE folder.parent = cte.id AND cte.depth < $2 AND NOT folder.trashed
)

SELECT id, name FROM cte WHERE depth = $2
`

func (s Store) RecursiveFolders(ctx context.Context, id string, depth int) (folders []ds.Folder, err error) {
	rows, err := s.DB.QueryContext(ctx, sqlRecursiveFolders, id, depth)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		f := ds.Folder{}
		err := rows.Scan(&f.ID, &f.Name)
		if err != nil {
			return nil, err
		}

		folders = append(folders, f)
	}

	return folders, rows.Err()
}
