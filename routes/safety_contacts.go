package routes

import (
	"net/http"

	"kabackend/database"
	"kabackend/models"
	"kabackend/utils"
)

func contactToOut(c *models.SafetyContact) SafetyContactOut {
	return SafetyContactOut{
		ID:           c.ID,
		Name:         c.Name,
		Relationship: c.Relationship,
		Phone:        c.Phone,
		Email:        c.Email,
		Address:      c.Address,
	}
}

func AddSafetyContactHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getCurrentUser(w, r)
	if !ok {
		return
	}

	var body SafetyContactCreate
	if err := decodeJSONBody(r, &body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid request body")
		return
	}
	var count int
	err := database.DB.QueryRow(`SELECT COUNT(*) FROM safety_contacts WHERE user_id = $1`, user.ID).Scan(&count)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if count >= 5 {
		writeError(w, http.StatusBadRequest, "You can add up to 5 safety contacts only.")
		return
	}

	var newID int64
	err = database.DB.QueryRow(
		`INSERT INTO safety_contacts (user_id, name, relationship, phone, email, address, created_time)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		user.ID, body.Name, body.Relationship, body.Phone, body.Email, body.Address, utils.PyUTCNowStr(),
	).Scan(&newID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	writeJSON(w, http.StatusOK, SafetyContactOut{
		ID: newID, Name: body.Name, Relationship: body.Relationship,
		Phone: body.Phone, Email: body.Email, Address: body.Address,
	})
}

func ListSafetyContactsHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getCurrentUser(w, r)
	if !ok {
		return
	}

	rows, err := database.DB.Query(
		`SELECT id, user_id, name, relationship, phone, email, address, created_time
		 FROM safety_contacts WHERE user_id = $1`,
		user.ID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	out := []SafetyContactOut{}
	for rows.Next() {
		var c models.SafetyContact
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Relationship, &c.Phone, &c.Email, &c.Address, &c.CreatedTime); err != nil {
			writeError(w, http.StatusInternalServerError, "Database error")
			return
		}
		out = append(out, contactToOut(&c))
	}

	writeJSON(w, http.StatusOK, out)
}

func UpdateSafetyContactHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getCurrentUser(w, r)
	if !ok {
		return
	}

	contactID, err := parsePathID(r, "id")
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid contact id")
		return
	}

	var body SafetyContactCreate
	if err := decodeJSONBody(r, &body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid request body")
		return
	}
	res, err := database.DB.Exec(
		`UPDATE safety_contacts SET name=$1, relationship=$2, phone=$3, email=$4, address=$5
		 WHERE id=$6 AND user_id=$7`,
		body.Name, body.Relationship, body.Phone, body.Email, body.Address, contactID, user.ID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeError(w, http.StatusNotFound, "Safety contact not found")
		return
	}

	writeJSON(w, http.StatusOK, SafetyContactOut{
		ID: contactID, Name: body.Name, Relationship: body.Relationship,
		Phone: body.Phone, Email: body.Email, Address: body.Address,
	})
}

func DeleteSafetyContactHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getCurrentUser(w, r)
	if !ok {
		return
	}

	contactID, err := parsePathID(r, "id")
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid contact id")
		return
	}

	res, err := database.DB.Exec(`DELETE FROM safety_contacts WHERE id=$1 AND user_id=$2`, contactID, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeError(w, http.StatusNotFound, "Safety contact not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Safety contact removed"})
}

// getSafetyContactsForUser mirrors the plain query used inside sos.py /
// notifications.py's helper functions.
func getSafetyContactsForUser(userID int64) ([]models.SafetyContact, error) {
	rows, err := database.DB.Query(
		`SELECT id, user_id, name, relationship, phone, email, address, created_time
		 FROM safety_contacts WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []models.SafetyContact
	for rows.Next() {
		var c models.SafetyContact
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Relationship, &c.Phone, &c.Email, &c.Address, &c.CreatedTime); err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, nil
}
