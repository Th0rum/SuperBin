/*
This file is part of GigaPaste.

GigaPaste is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

GigaPaste is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with GigaPaste. If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"app/internal/dbo"
)

func CheckExpiration(db *dbo.Queries) {
	// nit(reddec): to be replaced by context in parameters
	ctx := context.Background()
	for {
		err := db.WithTransaction(ctx, func(queries *dbo.Queries) error {
			return cleanupOldInTransaction(ctx, queries)
		})
		if err != nil {
			fmt.Println(err)
		}
		time.Sleep(10 * time.Second)
	}
}

func cleanupOldInTransaction(ctx context.Context, queries *dbo.Queries) error {
	expired, err := queries.ListExpired(ctx, strconv.FormatInt(time.Now().Unix(), 10))
	if err != nil {
		return fmt.Errorf("list expired: %w", err)
	}
	for _, v := range expired {
		err = queries.DeleteDataByID(ctx, v.ID)
		if err != nil {
			return fmt.Errorf("delete expired data %v: %w", v.ID, err)
		}

		err = os.Remove(v.Filepath)
		if err != nil {
			fmt.Println(err)
			// keep cleaning
		}
	}
	return nil
}
