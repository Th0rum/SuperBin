/*
This file is part of GigaPaste.

GigaPaste is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

GigaPaste is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with GigaPaste. If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Setting struct {
	FileSizeLimit                  int64  `json:"FileSizeLimitMB"`                //maximum allowed upload file size
	TextSizeLimit                  int64  `json:"TextSizeLimitMB"`                //maximum allowed upload text size
	StreamSizeLimit                int64  `json:"StreamSizeLimitKB"`              //streaming buffer size
	StreamThrottle                 int64  `json:"StreamThrottleMS"`               //streaming Sleep() timer to not use too much cpu
	Pbkdf2Iteraions                int    `json:"Pbkdf2Iteraions"`                //key derviation function iterations
	CmdUploadDefaultDurationMinute int64  `json:"CmdUploadDefaultDurationMinute"` //default file duration when uploaded through curl / other cmdline
	BaseUrl                        string `json:"BaseUrl"`                        //url prefix for reverse proxy setups (must include leading /)
}

var Global Setting

func InitSettings() {

	file, err := os.Open("./data/settings.json")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&Global)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	fmt.Printf("Settings loaded: %+v\n", Global)

}
