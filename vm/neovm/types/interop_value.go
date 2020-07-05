/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */

package types

import (
	"bytes"

	"github.com/xiemylogos/ontology/v2/vm/neovm/interfaces"
)

type InteropValue struct {
	Data interfaces.Interop
}

func NewInteropValue(value interfaces.Interop) InteropValue {
	return InteropValue{Data: value}
}

func (this *InteropValue) Equals(other InteropValue) bool {
	// todo: both nil?
	if this.Data == nil || other.Data == nil {
		return false
	}
	return bytes.Equal(this.Data.ToArray(), other.Data.ToArray())
}
