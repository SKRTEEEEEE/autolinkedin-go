# Usage Examples of New Topic Fields (Fase 2)

This document demonstrates how the new topic fields work with the prompt system and provide practical examples of their usage.

## Topic Structure Overview

```json
{
  "id": "unique_id",
  "user_id": "user_id",
  "name": "Topic Name",
  "description": "Human readable description",
  "prompt": "base1",                // References a prompt for idea generation
  "category": "Technology",         // Classification category
  "priority": 8,                    // 1-10 importance scale
  "ideas_count": 5,                 // Number of ideas to generate
  "keywords": ["tech", "software"], // Terms for {keywords} variable
  "related_topics": ["AI", "ML"],   // Related topics for {related_topics}
  "active": true,
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

## Prompt Variable Examples

### Basic Variable Substitution

**Topic:**
```json
{
  "name": "React Hooks",
  "prompt": "base1",
  "ideas_count": 5,
  "category": "Frontend",
  "priority": 8
}
```

**Prompt Template:**
```
Generate {ideas} ideas about {name} in category {category} with priority {priority}
```

**Result:**
```
Generate 5 ideas about React Hooks in category Frontend with priority 8
```

### Keywords Variable (`{keywords}`)

**Topic:**
```json
{
  "name": "Go Microservices",
  "keywords": ["gRPC", "Docker", "Kubernetes", "REST API", "gRPC Gateway"]
}
```

**Prompt Template:**
```
Generate innovative microservice ideas focusing on: {[keywords]}
```

**Result:**
```
Generate innovative microservice ideas focusing on: gRPC, Docker, Kubernetes, REST API, gRPC Gateway
```

### Related Topics Variable (`{[related_topics]}`)

**Topic:**
```json
{
  "name": "TypeScript Best Practices",
  "related_topics": ["JavaScript", "Node.js", "React", "Webpack"]
}
```

**Prompt Template:**
```
Create TypeScript tips that integrate with {[related_topics]}
```

**Result:**
```
Create TypeScript tips that integrate with JavaScript, Node.js, React, Webpack
```

### Complex Template with Multiple Variables

**Topic:**
```json
{
  "name": "DevOps Automation",
  "description": "CI/CD and deployment strategies",
  "prompt": "professional",
  "category": "DevOps",
  "priority": 9,
  "ideas_count": 7,
  "keywords": ["Jenkins", "GitLab", "Ansible", "Terraform"],
  "related_topics": ["Docker", "Kubernetes", "AWS"]
}
```

**Prompt Template:**
```
As a DevOps expert, generate {ideas} professional ideas about {name}.

Context:
- Category: {category}
- Priority Level: {priority}/10
- Focus Areas: {[keywords]}
- Integration with: {[related_topics]}

Requirements:
- Each idea should be actionable and specific
- Include implementation considerations
- Focus on automation and scalability
```

**Generated Prompt:**
```
As a DevOps expert, generate 7 professional ideas about DevOps Automation.

Context:
- Category: DevOps
- Priority Level: 9/10
- Focus Areas: Jenkins, GitLab, Ansible, Terraform
- Integration with: Docker, Kubernetes, AWS

Requirements:
- Each idea should be actionable and specific
- Include implementation considerations
- Focus on automation and scalability
```

## Prompt Types and Examples

### Idea Generation Prompts

#### Basic Idea Prompt (`base1`)
```
Generate {ideas} ideas about {name} focusing on {[keywords]}.
Each idea should be 10-200 characters and compelling for LinkedIn content.
```

#### Creative Ideas Prompt (`creative`)
```
Generate innovative {ideas} ideas about {name}.
Think outside the box and suggest unconventional approaches.
Target audience: professionals interested in {category}.
Keywords to incorporate: {[keywords]}.
Related areas to consider: {[related_topics]}.
```

#### Technical Ideas Prompt (`technical`)
```
Generate {ideas} technical ideas for {name} at priority {priority}.
Focus on implementation details and best practices.
Include specific technologies: {[keywords]}.
Consider integration with: {[related_topics]}.
```

### Draft Generation Prompts

#### Professional Draft Prompt
```
Create professional LinkedIn content based on:
Topic: {topic_name}
Content: {content}

User Context:
{user_context}

Requirements:
- Write in professional tone
- Include actionable insights
- Provide practical examples
```

## Example Workflows

### Workflow 1: Creating a Technical Topic

1. **Define the topic:**
```json
{
  "name": "GraphQL API Design",
  "description": "Best practices for GraphQL schema design and resolvers",
  "prompt": "technical",
  "category": "Backend",
  "priority": 8,
  "ideas_count": 5,
  "keywords": ["Apollo", "Resolvers", "Schema", "Subscriptions"],
  "related_topics": ["REST APIs", "Node.js", "PostgreSQL"]
}
```

2. **Generated template creates:**
```
Generate 5 technical ideas for GraphQL API Design at priority 8.
Focus on implementation details and best practices.
Include specific technologies: Apollo, Resolvers, Schema, Subscriptions.
Consider integration with: REST APIs, Node.js, PostgreSQL.
```

3. **Sample ideas generated:**
   - "Implementación de resolvers GraphQL con caching Redis"
   - "Patrones de diseño para esquemas GraphQL escalables"
   - "Subscriptions GraphQL con WebSocket y Socket.io"
   - "Optimización de consultas GraphQL con DataLoader"
   - "Migration strategies from REST to GraphQL"

### Workflow 2: Marketing Content Topic

1. **Define the topic:**
```json
{
  "name": "Content Marketing Strategy",
  "description": "Creating effective content for technical audiences",
  "prompt": "creative",
  "category": "Marketing",
  "priority": 7,
  "ideas_count": 8,
  "keywords": ["SEO", "Blog Posts", "Social Media", "Analytics"],
  "related_topics": ["Technical Writing", "Audience Building"]
}
```

2. **Generated template creates:**
```
Generate innovative 8 ideas about Content Marketing Strategy.
Think outside the box and suggest unconventional approaches.
Target audience: professionals interested in Marketing.
Keywords to incorporate: SEO, Blog Posts, Social Media, Analytics.
Related areas to consider: Technical Writing, Audience Building.
```

### Workflow 3: Draft Generation with User Context

1. **User with configuration:**
```json
{
  "name": "Ana Martínez",
  "configuration": {
    "name": "Ana Martínez",
    "expertise": "Cloud Architecture",
    "industry": "Technology",
    "tone_preference": "Technical"
  }
}
```

2. **Idea about Kubernetes:**
```json
{
  "content": "Implementación de patrones Circuit Breaker en microservicios con Istio",
  "topic_name": "Kubernetes Production"
}
```

3. **Generated prompt for drafts:**
```
Create professional LinkedIn content based on:
Topic: Kubernetes Production
Content: Implementación de patrones Circuit Breaker en microservicios con Istio

User Context:
Name: Ana Martínez
Expertise: Cloud Architecture
Tone: Technical

Requirements:
- Write in professional tone
- Include actionable insights
- Provide practical examples
```

## Best Practices

### 1. Topic Naming
- Use descriptive, searchable names
- Include technology keywords when relevant
- Avoid overly generic names

**Good examples:**
- "React Performance Optimization"
- "Kubernetes Security Best Practices"
- "Python Async Programming Patterns"

### 2. Keywords Selection
- Choose 3-7 relevant keywords
- Include technology terms and concepts
- Consider synonyms and related terms

**Example for TypeScript:**
```json
"keywords": ["TypeScript", "Types", "Generics", "Decorators", "Declaration Files"]
```

### 3. Related Topics
- Connect to complementary technologies
- Include both broader and related specific topics
- Avoid circular references

**Example for React:**
```json
"related_topics": ["JavaScript", "Redux", "Webpack", "Jest"]
```

### 4. Category and Priority
- Use consistent categories for better organization
- Set priority based on business importance (1-10)
- Higher priority topics might need more frequent idea generation

### 5. Prompt Selection
- Match prompt style to topic nature (technical vs. creative)
- Consider audience when choosing prompt type
- Test different prompts for similar topics to find best fit

## Migration from Old Structure

### Before Fase 2:
```json
{
  "name": "Python",
  "ideas": 3,
  // No prompt reference
}
```

### After Fase 2:
```json
{
  "name": "Python Programming",
  "description": "Python development best practices and patterns",
  "prompt": "base1",
  "category": "Programming",
  "priority": 7,
  "ideas_count": 3,      // Renamed from 'ideas'
  "keywords": ["Python", "Django", "Flask"],
  "related_topics": ["Data Science", "Web Development"]
}
```

This migration enables dynamic prompt generation with rich context and variable substitution.
