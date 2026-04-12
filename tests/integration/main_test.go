package integration

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Shared fixture state used across all test files.
var (
	coach1Token   string
	coach1ID      string
	coach1Login   string
	coach1Refresh string

	coach2Token string
	coach2ID    string
	coach2Login string

	athlete1Token   string
	athlete1ID      string
	athlete1Login   string
	athlete1Refresh string

	athlete2Token string
	athlete2ID    string
	athlete2Login string

	athlete3Token string
	athlete3ID    string
	athlete3Login string

	// IDs populated during tests
	connectionRequestID string // athlete3 → coach2
	groupID             string
	templateID          string
	assignmentID        string // for report/archive flow
)

func TestMain(m *testing.M) {
	runSuffix = fmt.Sprintf("%d", time.Now().UnixMilli())
	client = NewAPIClient(getBaseURL())

	// Health check with retry
	healthy := false
	for i := 0; i < 30; i++ {
		status, _, err := client.Get("/health", "")
		if err == nil && status == http.StatusOK {
			healthy = true
			break
		}
		time.Sleep(1 * time.Second)
	}
	if !healthy {
		log.Fatal("API gateway is not reachable at " + getBaseURL())
	}

	// Register coach1
	resp, _, err := registerUser(
		uniqueLogin("intcoach1"),
		uniqueEmail("intcoach1"),
		"Integration Coach One",
		"coach",
	)
	if err != nil {
		log.Fatalf("register coach1: %v", err)
	}
	coach1Token = resp.AccessToken
	coach1ID = resp.User.ID
	coach1Login = resp.User.Login
	coach1Refresh = resp.RefreshToken

	// Register coach2
	resp, _, err = registerUser(
		uniqueLogin("intcoach2"),
		uniqueEmail("intcoach2"),
		"Integration Coach Two",
		"coach",
	)
	if err != nil {
		log.Fatalf("register coach2: %v", err)
	}
	coach2Token = resp.AccessToken
	coach2ID = resp.User.ID
	coach2Login = resp.User.Login

	// Register athlete1
	resp, _, err = registerUser(
		uniqueLogin("intathlete1"),
		uniqueEmail("intathlete1"),
		"Integration Athlete One",
		"athlete",
	)
	if err != nil {
		log.Fatalf("register athlete1: %v", err)
	}
	athlete1Token = resp.AccessToken
	athlete1ID = resp.User.ID
	athlete1Login = resp.User.Login
	athlete1Refresh = resp.RefreshToken

	// Register athlete2
	resp, _, err = registerUser(
		uniqueLogin("intathlete2"),
		uniqueEmail("intathlete2"),
		"Integration Athlete Two",
		"athlete",
	)
	if err != nil {
		log.Fatalf("register athlete2: %v", err)
	}
	athlete2Token = resp.AccessToken
	athlete2ID = resp.User.ID
	athlete2Login = resp.User.Login

	// Register athlete3 (stays unconnected)
	resp, _, err = registerUser(
		uniqueLogin("intathlete3"),
		uniqueEmail("intathlete3"),
		"Integration Athlete Three",
		"athlete",
	)
	if err != nil {
		log.Fatalf("register athlete3: %v", err)
	}
	athlete3Token = resp.AccessToken
	athlete3ID = resp.User.ID
	athlete3Login = resp.User.Login

	// Wait for NATS events (user.registered -> user-service profile sync)
	time.Sleep(natsSyncDelay)

	// Connect athlete1 → coach1
	if err := connectAthleteToCoach(athlete1Token, coach1ID, coach1Token); err != nil {
		log.Fatalf("connect athlete1→coach1: %v", err)
	}

	// Connect athlete2 → coach1
	if err := connectAthleteToCoach(athlete2Token, coach1ID, coach1Token); err != nil {
		log.Fatalf("connect athlete2→coach1: %v", err)
	}

	os.Exit(m.Run())
}

// connectAthleteToCoach sends a connection request and accepts it.
func connectAthleteToCoach(athleteToken, coachID, coachToken string) error {
	// Send request
	status, data, err := client.Post("/api/v1/connections/request", map[string]string{
		"coach_id": coachID,
	}, athleteToken)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	if status != http.StatusCreated {
		return fmt.Errorf("send request returned %d: %s", status, string(data))
	}

	var cr ConnectionRequestResponse
	if err := json.Unmarshal(data, &cr); err != nil {
		return fmt.Errorf("unmarshal request: %w", err)
	}

	// Accept
	status, data, err = client.Put(
		fmt.Sprintf("/api/v1/connections/requests/%s/accept", cr.ID),
		nil,
		coachToken,
	)
	if err != nil {
		return fmt.Errorf("accept: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("accept returned %d: %s", status, string(data))
	}

	return nil
}

// TestIntegration is the single orchestrator that runs all subtests in order.
func TestIntegration(t *testing.T) {
	t.Run("Auth", func(t *testing.T) {
		t.Run("RegisterCoach_Success", testAuthRegisterCoachSuccess)
		t.Run("RegisterAthlete_Success", testAuthRegisterAthleteSuccess)
		t.Run("Register_DuplicateLogin", testAuthRegisterDuplicateLogin)
		t.Run("Register_LoginTooShort", testAuthRegisterLoginTooShort)
		t.Run("Register_LoginInvalidChars", testAuthRegisterLoginInvalidChars)
		t.Run("Register_InvalidEmail", testAuthRegisterInvalidEmail)
		t.Run("Register_PasswordTooShort", testAuthRegisterPasswordTooShort)
		t.Run("Register_InvalidRole", testAuthRegisterInvalidRole)
		t.Run("Register_EmptyBody", testAuthRegisterEmptyBody)
		t.Run("Login_Success", testAuthLoginSuccess)
		t.Run("Login_WrongPassword", testAuthLoginWrongPassword)
		t.Run("Login_NonexistentUser", testAuthLoginNonexistentUser)
		t.Run("Refresh_Success", testAuthRefreshSuccess)
		t.Run("Refresh_InvalidToken", testAuthRefreshInvalidToken)
	})

	t.Run("Profile", func(t *testing.T) {
		t.Run("GetMe_Coach", testProfileGetMeCoach)
		t.Run("GetMe_Athlete", testProfileGetMeAthlete)
		t.Run("GetMe_NoAuth", testProfileGetMeNoAuth)
		t.Run("Search_ByLogin", testProfileSearchByLogin)
		t.Run("Search_ByRole", testProfileSearchByRole)
		t.Run("Search_NoResults", testProfileSearchNoResults)
		t.Run("Search_Pagination", testProfileSearchPagination)
	})

	t.Run("Connection", func(t *testing.T) {
		t.Run("GetCoach_Connected", testConnectionGetCoachConnected)
		t.Run("GetCoach_Unconnected", testConnectionGetCoachUnconnected)
		t.Run("GetCoach_AsCoach", testConnectionGetCoachAsCoach)
		t.Run("GetAthletes_Coach", testConnectionGetAthletesCoach)
		t.Run("GetAthletes_AsAthlete", testConnectionGetAthletesAsAthlete)
		t.Run("Request_AlreadyHasCoach", testConnectionRequestAlreadyHasCoach)
		t.Run("Request_CoachSends", testConnectionRequestCoachSends)
		t.Run("Request_TargetNotCoach", testConnectionRequestTargetNotCoach)
		t.Run("Request_Success", testConnectionRequestSuccess)
		t.Run("Request_Duplicate", testConnectionRequestDuplicate)
		t.Run("Outgoing_Athlete", testConnectionOutgoingAthlete)
		t.Run("Outgoing_AsCoach", testConnectionOutgoingAsCoach)
		t.Run("Incoming_Coach", testConnectionIncomingCoach)
		t.Run("Incoming_AsAthlete", testConnectionIncomingAsAthlete)
		t.Run("Reject_Success", testConnectionRejectSuccess)
		t.Run("Reject_AlreadyRejected", testConnectionRejectAlreadyRejected)
	})

	t.Run("Group", func(t *testing.T) {
		t.Run("Create_Success", testGroupCreateSuccess)
		t.Run("Create_AsAthlete", testGroupCreateAsAthlete)
		t.Run("Create_EmptyName", testGroupCreateEmptyName)
		t.Run("List_Coach", testGroupListCoach)
		t.Run("Get_Success", testGroupGetSuccess)
		t.Run("Get_OtherCoach", testGroupGetOtherCoach)
		t.Run("Update_Success", testGroupUpdateSuccess)
		t.Run("Update_AsAthlete", testGroupUpdateAsAthlete)
		t.Run("AddMember_Athlete1", testGroupAddMemberAthlete1)
		t.Run("AddMember_Athlete2", testGroupAddMemberAthlete2)
		t.Run("AddMember_Duplicate", testGroupAddMemberDuplicate)
		t.Run("AddMember_Unconnected", testGroupAddMemberUnconnected)
		t.Run("Get_WithMembers", testGroupGetWithMembers)
		t.Run("Athlete_CanSeeGroup", testGroupAthleteCanSeeGroup)
		t.Run("NonMember_CannotSeeGroup", testGroupNonMemberCannotSeeGroup)
		t.Run("RemoveMember_Success", testGroupRemoveMemberSuccess)
		t.Run("RemoveMember_AlreadyRemoved", testGroupRemoveMemberAlreadyRemoved)
		t.Run("Delete_Success", testGroupDeleteSuccess)
	})

	t.Run("Template", func(t *testing.T) {
		t.Run("Create_Success", testTemplateCreateSuccess)
		t.Run("Create_AsAthlete", testTemplateCreateAsAthlete)
		t.Run("Create_MissingTitle", testTemplateCreateMissingTitle)
		t.Run("List_Success", testTemplateListSuccess)
		t.Run("List_AsAthlete", testTemplateListAsAthlete)
		t.Run("Get_Success", testTemplateGetSuccess)
		t.Run("Get_OtherCoach", testTemplateGetOtherCoach)
		t.Run("Get_NotFound", testTemplateGetNotFound)
		t.Run("Update_Success", testTemplateUpdateSuccess)
		t.Run("Update_EmptyBody", testTemplateUpdateEmptyBody)
		t.Run("Delete_Success", testTemplateDeleteSuccess)
		t.Run("Delete_NotFound", testTemplateDeleteNotFound)
	})

	t.Run("Training", func(t *testing.T) {
		t.Run("CreatePlan_WithAthleteIDs", testTrainingCreatePlanWithAthleteIDs)
		t.Run("CreatePlan_WithGroupID", testTrainingCreatePlanWithGroupID)
		t.Run("CreatePlan_WithSaveAsTemplate", testTrainingCreatePlanWithSaveAsTemplate)
		t.Run("CreatePlan_AsAthlete", testTrainingCreatePlanAsAthlete)
		t.Run("CreatePlan_NoTargets", testTrainingCreatePlanNoTargets)
		t.Run("CreatePlan_MissingTitle", testTrainingCreatePlanMissingTitle)
		t.Run("CreatePlan_InvalidDate", testTrainingCreatePlanInvalidDate)
		t.Run("GetAssignments_Athlete", testTrainingGetAssignmentsAthlete)
		t.Run("GetAssignments_Coach", testTrainingGetAssignmentsCoach)
		t.Run("GetAssignment_Success", testTrainingGetAssignmentSuccess)
		t.Run("GetAssignment_OtherAthlete", testTrainingGetAssignmentOtherAthlete)
		t.Run("GetAssignment_NotFound", testTrainingGetAssignmentNotFound)
		t.Run("SubmitReport_Success", testTrainingSubmitReportSuccess)
		t.Run("SubmitReport_AsCoach", testTrainingSubmitReportAsCoach)
		t.Run("SubmitReport_Duplicate", testTrainingSubmitReportDuplicate)
		t.Run("GetReport_AsAthlete", testTrainingGetReportAsAthlete)
		t.Run("GetReport_AsCoach", testTrainingGetReportAsCoach)
		t.Run("AssignmentStatus_Completed", testTrainingAssignmentStatusCompleted)
		t.Run("Archive_Success", testTrainingArchiveSuccess)
		t.Run("Archive_NotCompleted", testTrainingArchiveNotCompleted)
		t.Run("GetArchived", testTrainingGetArchived)
		t.Run("GetGroupPlans", testTrainingGetGroupPlans)
		t.Run("GetGroupPlans_AllIncludingPast", testTrainingGetGroupPlansAllIncludingPast)
		t.Run("GetGroupPlans_AsAthlete", testTrainingGetGroupPlansAsAthlete)
		t.Run("GetGroupPlans_EmptyResult", testTrainingGetGroupPlansEmptyResult)
		t.Run("DeleteAssignment_Success", testTrainingDeleteAssignmentSuccess)
	})

	t.Run("Notification", func(t *testing.T) {
		t.Run("Coach_HasNotifications", testNotificationCoachHas)
		t.Run("Athlete_HasNotifications", testNotificationAthleteHas)
		t.Run("Filter_Unread", testNotificationFilterUnread)
		t.Run("Pagination", testNotificationPagination)
		t.Run("NoAuth", testNotificationNoAuth)
		t.Run("MarkRead_Success", testNotificationMarkReadSuccess)
		t.Run("MarkRead_NotFound", testNotificationMarkReadNotFound)
		t.Run("MarkAllRead", testNotificationMarkAllRead)
		t.Run("DeviceToken_Success", testNotificationDeviceTokenSuccess)
		t.Run("DeviceToken_Empty", testNotificationDeviceTokenEmpty)
	})

	t.Run("Analytics", func(t *testing.T) {
		t.Run("Me_Summary", testAnalyticsMeSummary)
		t.Run("Me_Progress", testAnalyticsMeProgress)
		t.Run("Athlete_Summary_AsCoach", testAnalyticsAthleteSummaryAsCoach)
		t.Run("Athlete_Progress_AsCoach", testAnalyticsAthleteProgressAsCoach)
		t.Run("Overview", testAnalyticsOverview)
		t.Run("Overview_AsAthlete", testAnalyticsOverviewAsAthlete)
		t.Run("NoAuth", testAnalyticsNoAuth)
	})

	t.Run("AI", func(t *testing.T) {
		// Fast auth/role checks first, before heavy Ollama calls
		t.Run("Recommendations_AsAthlete", testAIRecommendationsAsAthlete)
		t.Run("Summary_AsAthlete", testAISummaryAsAthlete)
		t.Run("Recommendations_NoAuth", testAIRecommendationsNoAuth)
		t.Run("Recommendations_AsCoach", testAIRecommendationsAsCoach)
		t.Run("Analysis_AsCoach", testAIAnalysisAsCoach)
		t.Run("Summary_AsCoach", testAISummaryAsCoach)
	})
}

// requireStatus is a helper that checks HTTP status and prints body on failure.
func requireStatus(t *testing.T, expected, actual int, body []byte) {
	t.Helper()
	require.Equalf(t, expected, actual, "unexpected status code, body: %s", string(body))
}
