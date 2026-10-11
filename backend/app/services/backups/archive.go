package backups

import (
	"archive/tar"
	"errors"
	"io"
	"path"
	"strings"
)

// rewriteTar copies a tar stream, optionally stripping a leading path component
// and optionally adding one. Docker's copy API wraps directory contents in the
// directory entry itself, so archives round trip with a prefix change.
func rewriteTar(input io.Reader, output io.Writer, strip, add string) error {
	strip = strings.Trim(path.Clean("/"+strip), "/")
	add = strings.Trim(path.Clean("/"+add), "/")

	reader := tar.NewReader(input)
	writer := tar.NewWriter(output)

	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return errors.New("backups: the archive could not be read: " + err.Error())
		}

		name := path.Clean("/" + header.Name)

		if strip != "" {
			prefix := "/" + strip
			if name == prefix {
				continue
			}
			if !strings.HasPrefix(name, prefix+"/") {
				continue
			}

			name = strings.TrimPrefix(name, prefix)
		}

		name = strings.TrimPrefix(name, "/")
		if name == "" {
			continue
		}

		if add != "" {
			name = path.Join(add, name)
		}

		header.Name = name

		if header.Typeflag == tar.TypeLink {
			link := path.Clean("/" + header.Linkname)
			if strip != "" {
				link = strings.TrimPrefix(link, "/"+strip)
			}
			header.Linkname = strings.TrimPrefix(path.Join(add, strings.TrimPrefix(link, "/")), "/")
		}

		if err := writer.WriteHeader(header); err != nil {
			return errors.New("backups: the archive could not be written: " + err.Error())
		}

		if header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA {
			if _, err := io.Copy(writer, reader); err != nil {
				return errors.New("backups: the archive could not be written: " + err.Error())
			}
		}
	}

	return writer.Flush()
}
