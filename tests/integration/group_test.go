package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testGroupCreateSuccess(t *testing.T) {
	status, data, err := client.Post("/api/v1/groups", map[string]string{
		"name": "IntTest Group",
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, data)

	var group TrainingGroupDetail
	require.NoError(t, json.Unmarshal(data, &group))
	assert.NotEmpty(t, group.ID)
	assert.Equal(t, "IntTest Group", group.Name)

	groupID = group.ID
}

func testGroupCreateAsAthlete(t *testing.T) {
	status, data, err := client.Post("/api/v1/groups", map[string]string{
		"name": "Forbidden Group",
	}, athlete1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, data)
}

func testGroupCreateEmptyName(t *testing.T) {
	status, data, err := client.Post("/api/v1/groups", map[string]string{
		"name": "",
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, data)
}

func testGroupListCoach(t *testing.T) {
	require.NotEmpty(t, groupID, "depends on Create_Success")

	status, data, err := client.Get("/api/v1/groups", coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var resp PaginatedGroups
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.GreaterOrEqual(t, resp.Pagination.TotalItems, 1)
}

func testGroupGetSuccess(t *testing.T) {
	require.NotEmpty(t, groupID, "depends on Create_Success")

	status, data, err := client.Get(fmt.Sprintf("/api/v1/groups/%s", groupID), coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var group TrainingGroupDetail
	require.NoError(t, json.Unmarshal(data, &group))
	assert.Equal(t, groupID, group.ID)
	assert.Equal(t, "IntTest Group", group.Name)
	assert.Empty(t, group.Members)
}

func testGroupGetOtherCoach(t *testing.T) {
	require.NotEmpty(t, groupID, "depends on Create_Success")

	status, data, err := client.Get(fmt.Sprintf("/api/v1/groups/%s", groupID), coach2Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, data)
}

func testGroupUpdateSuccess(t *testing.T) {
	require.NotEmpty(t, groupID, "depends on Create_Success")

	status, data, err := client.Put(fmt.Sprintf("/api/v1/groups/%s", groupID), map[string]string{
		"name": "Updated Group Name",
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var group TrainingGroupDetail
	require.NoError(t, json.Unmarshal(data, &group))
	assert.Equal(t, "Updated Group Name", group.Name)
}

func testGroupUpdateAsAthlete(t *testing.T) {
	require.NotEmpty(t, groupID, "depends on Create_Success")

	status, data, err := client.Put(fmt.Sprintf("/api/v1/groups/%s", groupID), map[string]string{
		"name": "Hacked",
	}, athlete1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, data)
}

func testGroupAddMemberAthlete1(t *testing.T) {
	require.NotEmpty(t, groupID, "depends on Create_Success")

	status, data, err := client.Post(fmt.Sprintf("/api/v1/groups/%s/members", groupID), map[string]string{
		"athlete_id": athlete1ID,
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, data)

	var member GroupMemberResponse
	require.NoError(t, json.Unmarshal(data, &member))
	assert.Equal(t, athlete1ID, member.AthleteID)
	assert.NotEmpty(t, member.FullName)
	assert.NotEmpty(t, member.Login)
}

func testGroupAddMemberAthlete2(t *testing.T) {
	require.NotEmpty(t, groupID, "depends on Create_Success")

	status, data, err := client.Post(fmt.Sprintf("/api/v1/groups/%s/members", groupID), map[string]string{
		"athlete_id": athlete2ID,
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, data)
}

func testGroupAddMemberDuplicate(t *testing.T) {
	require.NotEmpty(t, groupID, "depends on AddMember_Athlete1")

	status, data, err := client.Post(fmt.Sprintf("/api/v1/groups/%s/members", groupID), map[string]string{
		"athlete_id": athlete1ID,
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusConflict, status, data)

	svcErr, err := parseServiceError(data)
	require.NoError(t, err)
	assert.Equal(t, "ALREADY_IN_GROUP", svcErr.Error.Code)
}

func testGroupAddMemberUnconnected(t *testing.T) {
	require.NotEmpty(t, groupID, "depends on Create_Success")

	status, data, err := client.Post(fmt.Sprintf("/api/v1/groups/%s/members", groupID), map[string]string{
		"athlete_id": athlete3ID,
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, data)
}

func testGroupGetWithMembers(t *testing.T) {
	require.NotEmpty(t, groupID, "depends on AddMember tests")

	status, data, err := client.Get(fmt.Sprintf("/api/v1/groups/%s", groupID), coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var group TrainingGroupDetail
	require.NoError(t, json.Unmarshal(data, &group))
	assert.Equal(t, 2, len(group.Members))
}

func testGroupAthleteCanSeeGroup(t *testing.T) {
	require.NotEmpty(t, groupID, "depends on AddMember_Athlete1")

	status, data, err := client.Get("/api/v1/groups", athlete1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var resp PaginatedGroups
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.GreaterOrEqual(t, resp.Pagination.TotalItems, 1)
}

func testGroupNonMemberCannotSeeGroup(t *testing.T) {
	require.NotEmpty(t, groupID, "depends on Create_Success")

	status, data, err := client.Get(fmt.Sprintf("/api/v1/groups/%s", groupID), athlete3Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, data)
}

func testGroupRemoveMemberSuccess(t *testing.T) {
	require.NotEmpty(t, groupID, "depends on AddMember_Athlete2")

	status, data, err := client.Delete(
		fmt.Sprintf("/api/v1/groups/%s/members/%s", groupID, athlete2ID),
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusNoContent, status, data)
}

func testGroupRemoveMemberAlreadyRemoved(t *testing.T) {
	require.NotEmpty(t, groupID, "depends on RemoveMember_Success")

	status, data, err := client.Delete(
		fmt.Sprintf("/api/v1/groups/%s/members/%s", groupID, athlete2ID),
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusNotFound, status, data)
}

func testGroupDeleteSuccess(t *testing.T) {
	// Create a separate group to delete (keep main group for training tests)
	status, data, err := client.Post("/api/v1/groups", map[string]string{
		"name": "Throwaway Group",
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, data)

	var group TrainingGroupDetail
	require.NoError(t, json.Unmarshal(data, &group))

	status, data, err = client.Delete(fmt.Sprintf("/api/v1/groups/%s", group.ID), coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusNoContent, status, data)

	// Verify it's gone
	status, data, err = client.Get(fmt.Sprintf("/api/v1/groups/%s", group.ID), coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusNotFound, status, data)
}
