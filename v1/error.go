package wsgraphql

import (
	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/graphql-go/graphql/language/location"
)

func wrapExtendedError(err error, loc []location.SourceLocation) error {
	_, ok := err.(gqlerrors.ExtendedError)
	if ok {
		return &gqlerrors.Error{
			Message:       err.Error(),
			OriginalError: err,
			Locations:     loc,
		}
	}

	return err
}

func formatError(err error) gqlerrors.FormattedError {
	var loc []location.SourceLocation

	fmterr, ok := err.(gqlerrors.FormattedError)
	if ok {
		err = fmterr.OriginalError()
		loc = fmterr.Locations
	}

	_, ok = err.(*gqlerrors.Error)
	if ok {
		return gqlerrors.FormatError(err)
	}

	return gqlerrors.FormatError(wrapExtendedError(err, loc))
}
