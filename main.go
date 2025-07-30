package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type Author struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	JobTitle    string             `bson:"job_title" json:"job_title"`
	Email       string             `bson:"email" json:"email"`
	LinkedinURL string             `bson:"linkedin_url" json:"linkedin_url"`
	GithubURL   string             `bson:"github_url" json:"github_url"`
	Hobbies     []string           `bson:"hobbies" json:"hobbies"`
}

// Project represents a project in the database
type Project struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name             string             `bson:"name" json:"name"`
	Category         string             `bson:"category" json:"category"`
	StartDate        time.Time          `bson:"start_date" json:"start_date"`
	EndDate          *time.Time         `bson:"end_date,omitempty" json:"end_date,omitempty"` // Pointer for nullable field
	Description      string             `bson:"description" json:"description"`
	AuthorID         primitive.ObjectID `bson:"author_id" json:"author_id"`
	TechnologiesUsed []string           `bson:"technologies_used" json:"technologies_used"`
	RepoURL          *string            `bson:"repo_url,omitempty" json:"repo_url,omitempty"` // Pointer for nullable field
}

// Contact represents contact information
type Contact struct {
	Phone string `bson:"phone" json:"phone"`
	Email string `bson:"email" json:"email"`
}

// Experience represents work experience
type Experience struct {
	JobTitle    string    `bson:"job_title" json:"job_title"`
	Company     string    `bson:"company" json:"company"`
	TimePresent int       `bson:"time_present" json:"time_present"` // in months
	Projects    []Project `bson:"projects" json:"projects"`
}

// Education represents educational background
type Education struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UniversityName string             `bson:"university_name" json:"university_name"`
	Major          string             `bson:"major" json:"major"`
	StartDate      time.Time          `bson:"start_date" json:"start_date"`
	EndDate        *time.Time         `bson:"end_date,omitempty" json:"end_date,omitempty"` // Pointer for nullable field
	Description    string             `bson:"description" json:"description"`
	StudentName    string             `bson:"student_name" json:"student_name"`
	StudentID      primitive.ObjectID `bson:"student_id" json:"student_id"`
}

// Resume represents a complete resume
type Resume struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Contact    Contact            `bson:"contact" json:"contact"`
	Experience []Experience       `bson:"experience" json:"experience"`
	Skills     []string           `bson:"skills" json:"skills"`
	Education  []Education        `bson:"education" json:"education"`
	AuthorID   primitive.ObjectID `bson:"author_id" json:"author_id"`
	AuthorName string             `bson:"author_name" json:"author_name"`
}

type APIHandler struct {
	service    *PortfolioService
	llmService *LLMService
}

// Database connection
func connectToMongoDB() (*mongo.Client, error) {
	// Get MongoDB connection string from environment variable
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		// Fallback to localhost for development
		mongoURI = "mongodb://localhost:27017"
		log.Println("MONGODB_URI not set, using localhost")
	}

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, err
	}

	// Test the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected to MongoDB!")
	return client, nil
}

// PortfolioService handles all database operations
type PortfolioService struct {
	client    *mongo.Client
	database  *mongo.Database
	authors   *mongo.Collection
	projects  *mongo.Collection
	resumes   *mongo.Collection
	education *mongo.Collection
}

// NewPortfolioService creates a new portfolio service instance
func NewPortfolioService(client *mongo.Client) *PortfolioService {
	// Get database name from environment variable or use default
	dbName := os.Getenv("MONGODB_DATABASE")
	if dbName == "" {
		dbName = "portfolio" // Default database name
	}

	db := client.Database(dbName)
	return &PortfolioService{
		client:    client,
		database:  db,
		authors:   db.Collection("authors"),
		projects:  db.Collection("projects"),
		resumes:   db.Collection("resumes"),
		education: db.Collection("education"),
	}
}

// Author query methods
func (ps *PortfolioService) GetAllAuthors(ctx context.Context) ([]Author, error) {
	cursor, err := ps.authors.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var authors []Author
	if err = cursor.All(ctx, &authors); err != nil {
		return nil, err
	}
	return authors, nil
}

func (ps *PortfolioService) GetAuthorByName(ctx context.Context, name string) (*Author, error) {
	var author Author
	filter := bson.M{"name": bson.M{"$regex": name, "$options": "i"}} // Case-insensitive search
	err := ps.authors.FindOne(ctx, filter).Decode(&author)
	if err != nil {
		return nil, err
	}
	return &author, nil
}

func (ps *PortfolioService) GetAuthorByEmail(ctx context.Context, email string) (*Author, error) {
	var author Author
	filter := bson.M{"email": email}
	err := ps.authors.FindOne(ctx, filter).Decode(&author)
	if err != nil {
		return nil, err
	}
	return &author, nil
}

func (ps *PortfolioService) GetAuthorByID(ctx context.Context, id primitive.ObjectID) (*Author, error) {
	var author Author
	filter := bson.M{"_id": id}
	err := ps.authors.FindOne(ctx, filter).Decode(&author)
	if err != nil {
		return nil, err
	}
	return &author, nil
}

func (ps *PortfolioService) CreateAuthor(ctx context.Context, author Author) (*Author, error) {
	result, err := ps.authors.InsertOne(ctx, author)
	if err != nil {
		return nil, err
	}
	author.ID = result.InsertedID.(primitive.ObjectID)
	return &author, nil
}

func (ps *PortfolioService) CountAuthors(ctx context.Context) (int64, error) {
	return ps.authors.CountDocuments(ctx, bson.M{})
}

// Project query methods
func (ps *PortfolioService) GetAllProjects(ctx context.Context) ([]Project, error) {
	cursor, err := ps.projects.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var projects []Project
	if err = cursor.All(ctx, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (ps *PortfolioService) GetProjectByName(ctx context.Context, name string) (*Project, error) {
	var project Project
	filter := bson.M{"name": bson.M{"$regex": name, "$options": "i"}}
	err := ps.projects.FindOne(ctx, filter).Decode(&project)
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (ps *PortfolioService) GetProjectsByCategory(ctx context.Context, category string) ([]Project, error) {
	cursor, err := ps.projects.Find(ctx, bson.M{"category": bson.M{"$regex": category, "$options": "i"}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var projects []Project
	if err = cursor.All(ctx, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (ps *PortfolioService) GetProjectsByAuthor(ctx context.Context, authorID primitive.ObjectID) ([]Project, error) {
	cursor, err := ps.projects.Find(ctx, bson.M{"author_id": authorID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var projects []Project
	if err = cursor.All(ctx, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (ps *PortfolioService) GetProjectsByTechnology(ctx context.Context, technology string) ([]Project, error) {
	cursor, err := ps.projects.Find(ctx, bson.M{"technologies_used": bson.M{"$regex": technology, "$options": "i"}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var projects []Project
	if err = cursor.All(ctx, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (ps *PortfolioService) CreateProject(ctx context.Context, project Project) (*Project, error) {
	result, err := ps.projects.InsertOne(ctx, project)
	if err != nil {
		return nil, err
	}
	project.ID = result.InsertedID.(primitive.ObjectID)
	return &project, nil
}

func (ps *PortfolioService) CountProjects(ctx context.Context) (int64, error) {
	return ps.projects.CountDocuments(ctx, bson.M{})
}

// Education query methods
func (ps *PortfolioService) GetAllEducation(ctx context.Context) ([]Education, error) {
	cursor, err := ps.education.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var education []Education
	if err = cursor.All(ctx, &education); err != nil {
		return nil, err
	}
	return education, nil
}

func (ps *PortfolioService) GetEducationByUniversity(ctx context.Context, university string) ([]Education, error) {
	cursor, err := ps.education.Find(ctx, bson.M{"university_name": bson.M{"$regex": university, "$options": "i"}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var education []Education
	if err = cursor.All(ctx, &education); err != nil {
		return nil, err
	}
	return education, nil
}

func (ps *PortfolioService) GetEducationByMajor(ctx context.Context, major string) ([]Education, error) {
	cursor, err := ps.education.Find(ctx, bson.M{"major": bson.M{"$regex": major, "$options": "i"}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var education []Education
	if err = cursor.All(ctx, &education); err != nil {
		return nil, err
	}
	return education, nil
}

func (ps *PortfolioService) GetEducationByStudent(ctx context.Context, studentID primitive.ObjectID) ([]Education, error) {
	cursor, err := ps.education.Find(ctx, bson.M{"student_id": studentID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var education []Education
	if err = cursor.All(ctx, &education); err != nil {
		return nil, err
	}
	return education, nil
}

func (ps *PortfolioService) CreateEducation(ctx context.Context, education Education) (*Education, error) {
	result, err := ps.education.InsertOne(ctx, education)
	if err != nil {
		return nil, err
	}
	education.ID = result.InsertedID.(primitive.ObjectID)
	return &education, nil
}

func (ps *PortfolioService) CountEducation(ctx context.Context) (int64, error) {
	return ps.education.CountDocuments(ctx, bson.M{})
}

// Resume query methods
func (ps *PortfolioService) GetAllResumes(ctx context.Context) ([]Resume, error) {
	cursor, err := ps.resumes.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var resumes []Resume
	if err = cursor.All(ctx, &resumes); err != nil {
		return nil, err
	}
	return resumes, nil
}

func (ps *PortfolioService) GetResumeByAuthor(ctx context.Context, authorID primitive.ObjectID) (*Resume, error) {
	var resume Resume
	filter := bson.M{"author_id": authorID}
	err := ps.resumes.FindOne(ctx, filter).Decode(&resume)
	if err != nil {
		return nil, err
	}
	return &resume, nil
}

func (ps *PortfolioService) GetResumesBySkill(ctx context.Context, skill string) ([]Resume, error) {
	cursor, err := ps.resumes.Find(ctx, bson.M{"skills": bson.M{"$regex": skill, "$options": "i"}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var resumes []Resume
	if err = cursor.All(ctx, &resumes); err != nil {
		return nil, err
	}
	return resumes, nil
}

func (ps *PortfolioService) CreateResume(ctx context.Context, resume Resume) (*Resume, error) {
	result, err := ps.resumes.InsertOne(ctx, resume)
	if err != nil {
		return nil, err
	}
	resume.ID = result.InsertedID.(primitive.ObjectID)
	return &resume, nil
}

func (ps *PortfolioService) CountResumes(ctx context.Context) (int64, error) {
	return ps.resumes.CountDocuments(ctx, bson.M{})
}

// Generic search method for LLM integration
func (ps *PortfolioService) SearchAll(ctx context.Context, query string) (map[string]interface{}, error) {
	results := make(map[string]interface{})

	// For broad queries, get all data. For specific queries, do targeted search
	/* isSpecificQuery := strings.Contains(strings.ToLower(query), "react") ||
		strings.Contains(strings.ToLower(query), "go") ||
		strings.Contains(strings.ToLower(query), "python") ||
		strings.Contains(strings.ToLower(query), "azure") ||
		strings.Contains(strings.ToLower(query), "salesforce") ||
		strings.Contains(strings.ToLower(query), "project") ||
		strings.Contains(strings.ToLower(query), "university") ||
		strings.Contains(strings.ToLower(query), "company")

	var authorFilter, projectFilter, educationFilter, resumeFilter bson.M

	if isSpecificQuery {
		// Use targeted search for specific terms
		authorFilter = bson.M{
			"$or": []bson.M{
				{"name": bson.M{"$regex": query, "$options": "i"}},
				{"job_title": bson.M{"$regex": query, "$options": "i"}},
				{"email": bson.M{"$regex": query, "$options": "i"}},
				{"linkedin_url": bson.M{"$regex": query, "$options": "i"}},
				{"github_url": bson.M{"$regex": query, "$options": "i"}},
				{"hobbies": bson.M{"$regex": query, "$options": "i"}},
			},
		}
		projectFilter = bson.M{
			"$or": []bson.M{
				{"name": bson.M{"$regex": query, "$options": "i"}},
				{"category": bson.M{"$regex": query, "$options": "i"}},
				{"description": bson.M{"$regex": query, "$options": "i"}},
				{"technologies_used": bson.M{"$regex": query, "$options": "i"}},
			},
		}
		educationFilter = bson.M{
			"$or": []bson.M{
				{"student_name": bson.M{"$regex": query, "$options": "i"}},
				{"university_name": bson.M{"$regex": query, "$options": "i"}},
				{"major": bson.M{"$regex": query, "$options": "i"}},
				{"description": bson.M{"$regex": query, "$options": "i"}},
			},
		}
		resumeFilter = bson.M{
			"$or": []bson.M{
				{"author_name": bson.M{"$regex": query, "$options": "i"}},
				{"skills": bson.M{"$regex": query, "$options": "i"}},
				{"experience.company": bson.M{"$regex": query, "$options": "i"}},
				{"experience.job_title": bson.M{"$regex": query, "$options": "i"}},
			},
		}
	} else {
		// For general queries, get all data
		authorFilter = bson.M{}
		projectFilter = bson.M{}
		educationFilter = bson.M{}
		resumeFilter = bson.M{}
	}
	*/

	var authorFilter, projectFilter, educationFilter, resumeFilter bson.M

	authorFilter = bson.M{}
	projectFilter = bson.M{}
	educationFilter = bson.M{}
	resumeFilter = bson.M{}

	// Search authors
	authors, _ := ps.authors.Find(ctx, authorFilter)
	var authorResults []Author
	authors.All(ctx, &authorResults)
	results["authors"] = authorResults
	authors.Close(ctx)

	// Search projects
	projects, _ := ps.projects.Find(ctx, projectFilter)
	var projectResults []Project
	projects.All(ctx, &projectResults)
	results["projects"] = projectResults
	projects.Close(ctx)

	// Search education
	education, _ := ps.education.Find(ctx, educationFilter)
	var educationResults []Education
	education.All(ctx, &educationResults)
	results["education"] = educationResults
	education.Close(ctx)

	// Search resumes
	resumes, _ := ps.resumes.Find(ctx, resumeFilter)
	var resumeResults []Resume
	resumes.All(ctx, &resumeResults)
	results["resumes"] = resumeResults
	resumes.Close(ctx)

	return results, nil
}

// LLMService handles OpenAI API interactions
type LLMService struct {
	client           openai.Client
	portfolioService *PortfolioService
}

// NewLLMService creates a new LLM service instance
func NewLLMService(apiKey string, portfolioService *PortfolioService) *LLMService {
	if apiKey == "" {
		log.Println("Warning: OpenAI API key not provided. Chatbot will be disabled.")
		return nil
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &LLMService{
		client:           client,
		portfolioService: portfolioService,
	}
}

// ProcessQuery handles user queries with portfolio context
func (l *LLMService) ProcessQuery(ctx context.Context, query string) (string, error) {
	if l == nil {
		return "Chatbot is not available. OpenAI API key not configured.", nil
	}

	log.Printf("Processing chatbot query: %s", query)

	// Get relevant portfolio data as context
	searchResults, err := l.portfolioService.SearchAll(ctx, query)
	if err != nil {
		log.Printf("Error searching portfolio data: %v", err)
		return "", fmt.Errorf("failed to search portfolio data: %w", err)
	}

	// Log what data we found
	log.Printf("Search results for query '%s':", query)
	for collection, data := range searchResults {
		if dataSlice, ok := data.([]interface{}); ok {
			log.Printf("  %s: %d items", collection, len(dataSlice))
		} else if dataSlice, ok := data.([]Author); ok {
			log.Printf("  %s: %d authors", collection, len(dataSlice))
		} else if dataSlice, ok := data.([]Project); ok {
			log.Printf("  %s: %d projects", collection, len(dataSlice))
		} else if dataSlice, ok := data.([]Education); ok {
			log.Printf("  %s: %d education", collection, len(dataSlice))
		} else if dataSlice, ok := data.([]Resume); ok {
			log.Printf("  %s: %d resumes", collection, len(dataSlice))
		}
	}

	// Convert search results to JSON for context
	contextData, err := json.MarshalIndent(searchResults, "", "  ")
	if err != nil {
		log.Printf("Error marshaling context data: %v", err)
		return "", fmt.Errorf("failed to marshal context data: %w", err)
	}

	// Limit context size to prevent token overflow
	contextString := string(contextData)
	if len(contextString) > 8000 {
		contextString = contextString[:8000] + "...[truncated]"
		log.Printf("Context truncated to 8000 characters")
	}

	log.Printf("Context data being sent to OpenAI (first 500 chars): %s", contextString[:min(500, len(contextString))])

	// Create a comprehensive prompt with portfolio context
	prompt := fmt.Sprintf(`You are BILLIEBOT, a professional portfolio assistant for Billie Mallady, a talented Software Engineer. You have access to Billie's complete portfolio data in the form of MongoDB documents including projects, work experience, education, and skills, resume and hobbies. The following data structures apply:


	AUTHORS:
	Here you will find information about Billie Mallady, including their name, job title, email, LinkedIn URL, GitHub URL, and hobbies.

	PROJECTS:
	Here you will find information about Billie's projects, including project names, descriptions, technologies used, and links to live demos or repositories (if availiable). 

	EDUCATION:
	Here you will find information about Billie's education, including university name, field of study and start and end dates. 

	RESUMES:
	Here you will find information about Billie's resume, including contact information, work experience, skills, and education.



	PORTFOLIO DATA:
		%s

		USER QUESTION: %s

		Instructions:
		- Answer questions about Billie's professional background, projects, skills, and experience
		- Be conversational but professional
		- Do not assume that Billie knows programming languages or technologies not referenced in their portfolio. 
		- If the question is about specific projects, provide detailed information including technologies used
		- If asked about skills or experience, reference specific examples from the work history, and present in bullet points if you can
		- If the question isn't related to Billie's portfolio, politely redirect to professional topics.
		- Do not lie about Billie or provide false information.
		- Keep responses concise but informative
		- Use a friendly, confident tone that reflects Billie's professional capabilities
		- Include relevant examples from the portfolio data to support your answers

		Please provide a helpful response based on the portfolio data above.
		Provide your response separated by newline characters where appropriate.

`, contextString, query)

	log.Printf("Sending request to OpenAI...")

	// Send request to OpenAI using the official client (corrected syntax)
	completion, err := l.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
		Model: openai.ChatModelGPT3_5Turbo,
	})

	if err != nil {
		log.Printf("OpenAI API error: %v", err)
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(completion.Choices) == 0 {
		log.Printf("No choices returned from OpenAI")
		return "I'm sorry, I couldn't generate a response. Please try again.", nil
	}

	response := completion.Choices[0].Message.Content
	log.Printf("OpenAI response received: %d characters", len(response))

	return response, nil
}

// HTTP Handlers

func NewAPIHandler(service *PortfolioService, llmService *LLMService) *APIHandler {
	return &APIHandler{
		service:    service,
		llmService: llmService,
	}
}

// CORS middleware
func (h *APIHandler) enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// Authors endpoints
func (h *APIHandler) handleAuthors(w http.ResponseWriter, r *http.Request) {
	h.enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()

	// Check for query parameters
	name := r.URL.Query().Get("name")
	email := r.URL.Query().Get("email")

	if name != "" {
		author, err := h.service.GetAuthorByName(ctx, name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]*Author{author})
		return
	}

	if email != "" {
		author, err := h.service.GetAuthorByEmail(ctx, email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]*Author{author})
		return
	}

	// Get all authors
	authors, err := h.service.GetAllAuthors(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authors)
}

func (h *APIHandler) handleAuthorsCount(w http.ResponseWriter, r *http.Request) {
	h.enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	ctx := context.Background()
	count, err := h.service.CountAuthors(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"count": count})
}

// Projects endpoints
func (h *APIHandler) handleProjects(w http.ResponseWriter, r *http.Request) {
	h.enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()

	// Check for query parameters
	name := r.URL.Query().Get("name")
	category := r.URL.Query().Get("category")
	technology := r.URL.Query().Get("technology")
	authorIDStr := r.URL.Query().Get("author_id")

	if name != "" {
		project, err := h.service.GetProjectByName(ctx, name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]*Project{project})
		return
	}

	if category != "" {
		projects, err := h.service.GetProjectsByCategory(ctx, category)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(projects)
		return
	}

	if technology != "" {
		projects, err := h.service.GetProjectsByTechnology(ctx, technology)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(projects)
		return
	}

	if authorIDStr != "" {
		authorID, err := primitive.ObjectIDFromHex(authorIDStr)
		if err != nil {
			http.Error(w, "Invalid author ID", http.StatusBadRequest)
			return
		}
		projects, err := h.service.GetProjectsByAuthor(ctx, authorID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(projects)
		return
	}

	// Get all projects
	projects, err := h.service.GetAllProjects(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

func (h *APIHandler) handleProjectsCount(w http.ResponseWriter, r *http.Request) {
	h.enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	ctx := context.Background()
	count, err := h.service.CountProjects(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"count": count})
}

// Education endpoints
func (h *APIHandler) handleEducation(w http.ResponseWriter, r *http.Request) {
	h.enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()

	// Check for query parameters
	university := r.URL.Query().Get("university")
	major := r.URL.Query().Get("major")
	studentIDStr := r.URL.Query().Get("student_id")

	if university != "" {
		education, err := h.service.GetEducationByUniversity(ctx, university)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(education)
		return
	}

	if major != "" {
		education, err := h.service.GetEducationByMajor(ctx, major)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(education)
		return
	}

	if studentIDStr != "" {
		studentID, err := primitive.ObjectIDFromHex(studentIDStr)
		if err != nil {
			http.Error(w, "Invalid student ID", http.StatusBadRequest)
			return
		}
		education, err := h.service.GetEducationByStudent(ctx, studentID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(education)
		return
	}

	// Get all education
	education, err := h.service.GetAllEducation(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(education)
}

func (h *APIHandler) handleEducationCount(w http.ResponseWriter, r *http.Request) {
	h.enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	ctx := context.Background()
	count, err := h.service.CountEducation(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"count": count})
}

// Resumes endpoints
func (h *APIHandler) handleResumes(w http.ResponseWriter, r *http.Request) {
	h.enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()

	// Check for query parameters
	authorIDStr := r.URL.Query().Get("author_id")
	skill := r.URL.Query().Get("skill")

	if authorIDStr != "" {
		authorID, err := primitive.ObjectIDFromHex(authorIDStr)
		if err != nil {
			http.Error(w, "Invalid author ID", http.StatusBadRequest)
			return
		}
		resume, err := h.service.GetResumeByAuthor(ctx, authorID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]*Resume{resume})
		return
	}

	if skill != "" {
		resumes, err := h.service.GetResumesBySkill(ctx, skill)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resumes)
		return
	}

	// Get all resumes
	resumes, err := h.service.GetAllResumes(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resumes)
}

func (h *APIHandler) handleResumesCount(w http.ResponseWriter, r *http.Request) {
	h.enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	ctx := context.Background()
	count, err := h.service.CountResumes(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"count": count})
}

// Search endpoint for LLM integration
func (h *APIHandler) handleSearch(w http.ResponseWriter, r *http.Request) {
	h.enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	results, err := h.service.SearchAll(ctx, query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// Chatbot endpoint
func (h *APIHandler) handleChatbot(w http.ResponseWriter, r *http.Request) {
	// Add recovery to prevent server crashes
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Chatbot handler panic: %v", r)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}()

	h.enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Query string `json:"query"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Printf("Error decoding chatbot request: %v", err)
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(request.Query) == "" {
		http.Error(w, "Query cannot be empty", http.StatusBadRequest)
		return
	}

	log.Printf("Chatbot request received: %s", request.Query)

	if h.llmService == nil {
		log.Printf("LLM service is nil, chatbot disabled")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"response": "Sorry, the chatbot is currently unavailable. Please ensure OPENAI_API_KEY is configured.",
			"query":    request.Query,
		})
		return
	}

	ctx := context.Background()
	response, err := h.llmService.ProcessQuery(ctx, request.Query)
	if err != nil {
		log.Printf("Error processing chatbot query: %v", err)
		http.Error(w, fmt.Sprintf("Chatbot error: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Chatbot response generated successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"response": response,
		"query":    request.Query,
	})
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Connect to MongoDB
	client, err := connectToMongoDB()
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer client.Disconnect(context.TODO())

	// Create portfolio service
	service := NewPortfolioService(client)

	// Create LLM service (will be nil if API key not provided)

	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	llmService := NewLLMService(openaiAPIKey, service)

	// Create API handler
	handler := NewAPIHandler(service, llmService)

	// Setup routes
	http.HandleFunc("/api/authors", handler.handleAuthors)
	http.HandleFunc("/api/authors/count", handler.handleAuthorsCount)
	http.HandleFunc("/api/projects", handler.handleProjects)
	http.HandleFunc("/api/projects/count", handler.handleProjectsCount)
	http.HandleFunc("/api/education", handler.handleEducation)
	http.HandleFunc("/api/education/count", handler.handleEducationCount)
	http.HandleFunc("/api/resumes", handler.handleResumes)
	http.HandleFunc("/api/resumes/count", handler.handleResumesCount)
	http.HandleFunc("/api/search", handler.handleSearch)
	http.HandleFunc("/api/chatbot", handler.handleChatbot)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Portfolio API server starting on port %s\n", port)
	fmt.Println("Available endpoints:")
	fmt.Println("  GET  /api/authors")
	fmt.Println("  GET  /api/authors/count")
	fmt.Println("  GET  /api/projects")
	fmt.Println("  GET  /api/projects/count")
	fmt.Println("  GET  /api/education")
	fmt.Println("  GET  /api/education/count")
	fmt.Println("  GET  /api/resumes")
	fmt.Println("  GET  /api/resumes/count")
	fmt.Println("  GET  /api/search?q=<query>")
	fmt.Println("  POST /api/chatbot (JSON: {\"query\": \"your question\"})")
	fmt.Println("\nQuery parameters supported:")
	fmt.Println("  Authors: ?name=<name>, ?email=<email>")
	fmt.Println("  Projects: ?name=<name>, ?category=<category>, ?technology=<tech>, ?author_id=<id>")
	fmt.Println("  Education: ?university=<university>, ?major=<major>, ?student_id=<id>")
	fmt.Println("  Resumes: ?author_id=<id>, ?skill=<skill>")

	if llmService != nil {
		fmt.Println("\n🤖 Chatbot is ENABLED with OpenAI integration")
	} else {
		fmt.Println("\n⚠️  Chatbot is DISABLED (set OPENAI_API_KEY environment variable to enable)")
	}

	fmt.Println("\nNOTE: All endpoints except chatbot are read-only. No create/update/delete operations are available.")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
