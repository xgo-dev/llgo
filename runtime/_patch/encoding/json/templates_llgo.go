//go:build llgo

package json

//llgointernal:template:bind Marshal json op=marshal value=v
//llgointernal:template:bind Unmarshal json op=unmarshal input=data value=v
//llgointernal:template:bind MarshalIndent json op=fallback
//llgointernal:template:bind NewEncoder json op=fallback
//llgointernal:template:bind NewDecoder json op=fallback
