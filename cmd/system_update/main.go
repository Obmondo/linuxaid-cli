package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/bitfield/script"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	obmondoAPIURL = "https://api.obmondo.com/api"
)
