package vault

import (
	"bytes"
	"testing"
)

func TestSealRequiresKeyAndAssociatedData(t *testing.T) {
	key, err := randomKey()
	if err != nil {
		t.Fatal(err)
	}
	other, err := randomKey()
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := seal(key, []byte("client ciphertext"), itemAAD(7, "one"))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := open(key, ciphertext, itemAAD(7, "one"))
	if err != nil || !bytes.Equal(plain, []byte("client ciphertext")) {
		t.Fatalf("open failed: %v", err)
	}
	for _, test := range []struct {
		name string
		key  []byte
		aad  string
	}{
		{"wrong key", other, itemAAD(7, "one")},
		{"wrong user", key, itemAAD(8, "one")},
		{"wrong item", key, itemAAD(7, "two")},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := open(test.key, ciphertext, test.aad); err == nil {
				t.Fatal("decryption should fail")
			}
		})
	}
}
