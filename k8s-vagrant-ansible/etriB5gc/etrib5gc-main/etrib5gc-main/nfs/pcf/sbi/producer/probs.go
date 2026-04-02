package producer

import (
	"github.com/reogac/sbi/models"
	"net/http"
)

var AmPolNotCreatedProb *models.ProblemDetails = &models.ProblemDetails{
	Status: http.StatusForbidden,
	Detail: "AmPol not created",
}

var NoAmPolProb *models.ProblemDetails = &models.ProblemDetails{
	Status: http.StatusNotFound,
	Detail: "AmPol not found",
}

var SmPolNotCreatedProb *models.ProblemDetails = &models.ProblemDetails{
	Status: http.StatusForbidden,
	Detail: "SmPol not created",
}

var NoSmPolProb *models.ProblemDetails = &models.ProblemDetails{
	Status: http.StatusNotFound,
	Detail: "SmPol not found",
}

var InternalProb *models.ProblemDetails = &models.ProblemDetails{
	Status: http.StatusInternalServerError,
	Detail: "Internal error",
}
