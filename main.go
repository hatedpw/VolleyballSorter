package main

import (
	"encoding/csv"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

const membersPerTeam = 5
const maxTeams = 6

type Person struct {
	FirstName string
	Surname   string
	Committee bool
	Grade     string
}

func main() {
	// Read the eventbrite.csv file
	eventbriteParticipants, err := readCSV("eventbrite.csv")
	if err != nil {
		log.Fatalf("Failed to read eventbrite.csv: %v", err)
	}
	// Assume the first row of eventbrite.csv is the header and skip it
	eventbriteParticipants = eventbriteParticipants[1:]

	// Create a map to hold grading information for participants from eventbrite
	gradeMap := make(map[string]Person)
	for _, p := range eventbriteParticipants {
		fullName := p[3] + " " + p[4]
		gradeMap[fullName] = Person{
			FirstName: p[3],
			Surname:   p[4],
			Committee: false, // Default to false, will update later if needed
			Grade:     "c",   // Default to 'c', will update later if needed
		}
	}

	// Read the existing spikersgrading.csv file
	existingGrading, err := readCSV("spikersgrading.csv")
	if err != nil {
		log.Fatalf("Failed to read spikersgrading.csv: %v", err)
	}
	// Assume the first row of spikersgrading.csv is the header and skip it
	existingGradingHeaders := existingGrading[0] // Store headers for later
	existingGrading = existingGrading[1:]        // Skip headers for processing

	// Update gradeMap with existing grading information
	for _, gradeInfo := range existingGrading {
		fullName := gradeInfo[0] + " " + gradeInfo[1]
		if person, exists := gradeMap[fullName]; exists {
			// Update existing entry with grade and committee status
			person.Committee = strings.ToLower(gradeInfo[2]) == "true"
			person.Grade = strings.ToLower(gradeInfo[3])
			gradeMap[fullName] = person
		}
		// If the person is not in the eventbrite list, they are not added to gradeMap, thus not to the teams
	}
	println("map created")
	// Determine grades and committee status for eventbrite participants
	for _, p := range eventbriteParticipants {
		fullName := p[3] + " " + p[4]
		if _, exists := gradeMap[fullName]; !exists {
			gradeMap[fullName] = Person{
				FirstName: p[3],
				Surname:   p[4],
				Committee: false,
				Grade:     "c",
			}
		}
	}

	// Create participants list from the gradeMap to include all graded participants
	var participants []Person
	for _, person := range gradeMap {
		participants = append(participants, person)
	}

	// Sort participants by grade to make the distribution easier
	sort.Slice(participants, func(i, j int) bool {
		if participants[i].Grade == participants[j].Grade {
			return participants[i].Committee && !participants[j].Committee
		}
		return participants[i].Grade < participants[j].Grade
	})
	var numTeams int
	switch {
	case len(participants) >= 30:
		numTeams = 6
	case len(participants) >= 24:
		numTeams = 4
	default:
		numTeams = len(participants) / membersPerTeam
		if len(participants)%membersPerTeam != 0 {
			numTeams++ // If there's a remainder, add one more team
		}
	}

	teams := distributeParticipants(participants, numTeams)
	if err := writeTeamsCSV(teams, "teams.csv"); err != nil {
		log.Fatalf("Failed to write teams.csv: %v", err)
	}
	if err := writeGradingCSV(gradeMap, existingGradingHeaders, "spikersgrading.csv"); err != nil {
		log.Fatalf("Failed to write updated spikersgrading.csv: %v", err)
	}
}

// readCSV reads a CSV file and returns a slice of records
func readCSV(filename string) ([][]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	r := csv.NewReader(file)
	r.Comma = ','
	r.Comment = '#'
	r.TrimLeadingSpace = true

	var records [][]string
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	println("read csv 2")
	return records, nil
}

// distributeParticipants organizes participants into teams
func distributeParticipants(participants []Person, maxTeams int) [][]Person {
	// Create teams
	teams := make([][]Person, maxTeams)
	for i := range teams {
		teams[i] = make([]Person, 0)
	}

	// Ensure every team has at least one committee member
	for i := 0; i < maxTeams; i++ {
		for j, participant := range participants {
			if participant.Committee {
				teams[i] = append(teams[i], participant)
				participants = append(participants[:j], participants[j+1:]...)
				break
			}
		}
	}

	// Now distribute remaining participants ensuring grade criteria
	grades := []string{"a", "b", "c"}
	for _, grade := range grades {
		// Try to place participants of this grade into teams
		for len(participants) > 0 {
			placedInThisRound := false
			for i, team := range teams {
				if len(team) >= membersPerTeam {
					continue
				}
				for j, participant := range participants {
					if participant.Grade == grade {
						teams[i] = append(teams[i], participant)
						participants = append(participants[:j], participants[j+1:]...)
						placedInThisRound = true
						break
					}
				}
				if len(participants) == 0 {
					break
				}
			}
			if !placedInThisRound {
				break
			}
		}
	}

	// Fill any remaining spots with any remaining participants by rotating through the teams
	teamIndex := 0 // Start with the first team
	for len(participants) > 0 {
		// Append the next participant to the current team
		teams[teamIndex] = append(teams[teamIndex], participants[0])
		// Remove the placed participant from the slice
		participants = participants[1:]
		// Move to the next team
		teamIndex = (teamIndex + 1) % maxTeams // Ensure the team index wraps around
	}

	return teams
}

// writeTeamsCSV writes the teams to a CSV file
func writeTeamsCSV(teams [][]Person, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for i, team := range teams {
		// Start each team on a new row
		if err := writer.Write([]string{"Team " + strconv.Itoa(i+1)}); err != nil {
			return err
		}
		for _, person := range team {
			record := []string{person.FirstName, person.Surname, strconv.FormatBool(person.Committee), person.Grade}
			if err := writer.Write(record); err != nil {
				return err
			}
		}
	}
	println("writing teams to csv")
	return nil
}

func writeGradingCSV(gradeMap map[string]Person, existingGradingHeaders []string, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the header
	if err := writer.Write(existingGradingHeaders); err != nil {
		return err
	}

	// Go through the gradeMap and write entries
	for _, person := range gradeMap {
		record := []string{person.FirstName, person.Surname, strconv.FormatBool(person.Committee), strings.ToUpper(person.Grade)}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}
