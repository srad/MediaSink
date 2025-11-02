# Video Analysis API Documentation

## Overview

The Video Analysis API provides automated detection of scenes and highlights in video recordings. Analysis runs automatically after preview frames are generated, and results can be retrieved via the API.

---

## API Endpoints

### Get Analysis Results

**Endpoint:** `GET /api/v1/analysis/{recordingId}`

**Parameters:**
- `recordingId` (path, required, uint): The recording ID to get analysis for

**Response:**

```json
{
  "analysisId": 5,
  "recordingId": 123,
  "status": "completed",
  "scenes": [
    {
      "startTime": 0.0,
      "endTime": 45.5,
      "changeIntensity": 0.85
    },
    {
      "startTime": 45.5,
      "endTime": 120.2,
      "changeIntensity": 0.92
    }
  ],
  "highlights": [
    {
      "timestamp": 2.5,
      "intensity": 0.65,
      "type": "motion"
    },
    {
      "timestamp": 15.3,
      "intensity": 0.78,
      "type": "motion"
    },
    {
      "timestamp": 67.2,
      "intensity": 0.55,
      "type": "motion"
    }
  ]
}
```

**HTTP Status Codes:**
- `200 OK`: Analysis results returned successfully
- `400 Bad Request`: Invalid recording ID format
- `404 Not Found`: Recording not found
- `500 Internal Server Error`: Server error

---

## Response Field Documentation

### Top-Level Fields

#### `analysisId` (uint | null)

**Type:** Unsigned integer or null

**Description:** Unique identifier for the analysis record in the database.

**Meaning:**
- `null`: No analysis has been performed for this recording yet (not analyzed)
- `number`: Analysis record exists with the given ID

---

#### `recordingId` (uint)

**Type:** Unsigned integer (required)

**Description:** The ID of the recording being analyzed. Always present in the response.

**Range:** > 0

---

#### `status` (string | null)

**Type:** String or null

**Description:** The current status of the analysis job.

**Possible Values:**
- `null`: No analysis record exists (recording not yet analyzed)
- `"pending"`: Analysis job is queued, waiting to start
- `"processing"`: Analysis is currently running
- `"completed"`: Analysis finished successfully and results are available
- `"error"`: Analysis failed (check error logs on server)

**Interpretation:**
- When `null`: Call the analysis endpoint to trigger analysis
- When `"pending"` or `"processing"`: Poll this endpoint later to get results
- When `"completed"`: `scenes` and `highlights` arrays contain valid data
- When `"error"`: Analysis failed; results may not be available

---

### Scenes Array

#### `scenes` (array of SceneInfo)

**Type:** Array of objects

**Description:** List of detected scene boundaries in the video.

**When Empty:**
- Recording has not been analyzed yet (status is null)
- No significant scene changes were detected in the video

**SceneInfo Object:**

##### `startTime` (float)

**Type:** Floating-point number

**Unit:** Seconds

**Description:** The timestamp where the scene starts.

**Range:** 0.0 to video duration

---

##### `endTime` (float)

**Type:** Floating-point number

**Unit:** Seconds

**Description:** The timestamp where the scene ends.

**Range:** > startTime, up to video duration

---

##### `changeIntensity` (float)

**Type:** Floating-point number (0-1 scale)

**Description:** Measures how dramatic the scene change is at the scene boundary.

**Range:** 0.0 to 1.0

**Interpretation:**

| Value Range | Meaning | What It Indicates |
|-------------|---------|-------------------|
| 0.0 - 0.3 | Subtle transition | Minor variations in lighting, slow camera movement, gradual fades |
| 0.3 - 0.6 | Moderate change | Normal cuts between scenes, typical scene transitions |
| 0.6 - 0.85 | Significant change | Abrupt cuts, major camera movements, significant lighting changes |
| 0.85 - 1.0 | Very dramatic change | Complete scene changes, hard cuts, major compositional shifts |

**Technical Details:**
- Based on Structural Similarity Index (SSIM) comparison between consecutive frames
- Higher values indicate greater perceptual difference between frames
- Calculated as: `1.0 - SSIM_value` where SSIM ranges from 0 (completely different) to 1 (identical)

---

### Highlights Array

#### `highlights` (array of HighlightInfo)

**Type:** Array of objects

**Description:** List of moments with detected motion or activity above threshold.

**When Empty:**
- Recording has not been analyzed yet (status is null)
- No significant motion was detected in the video

**HighlightInfo Object:**

##### `timestamp` (float)

**Type:** Floating-point number

**Unit:** Seconds

**Description:** The exact timestamp where the highlight occurs (frame with detected motion).

**Range:** 0.0 to video duration

---

##### `intensity` (float)

**Type:** Floating-point number (0-1 scale)

**Description:** Measures the amount of motion or activity at this moment.

**Range:** 0.0 to 1.0

**Interpretation:**

| Value Range | Motion Level | What It Indicates |
|-------------|--------------|-------------------|
| 0.0 - 0.2 | Minimal | Almost static scene, very little to no movement |
| 0.2 - 0.5 | Low-Moderate | Gentle movements, slow motion, minor activity |
| 0.5 - 0.75 | Moderate-High | Active scene, clear movement, noticeable activity |
| 0.75 - 1.0 | Very High | Intense motion, fast movement, significant activity |

**Technical Details:**
- Based on pixel-level RGB difference between consecutive frames
- Threshold: Motion is only recorded if magnitude >= 0.05 (5%)
- Normalized to 0-1 range for standardized comparison

---

##### `type` (string)

**Type:** String

**Description:** Category of the highlight.

**Current Values:**
- `"motion"`: Motion/activity detection (frame-to-frame pixel changes)

**Future Extensions:**
Other types may be added (e.g., "scene_change", "flash", "transition")

---

## Data Structure Summary

```
AnalysisResponse {
  analysisId?:  uint | null           // null = not analyzed
  recordingId:  uint                   // always present (required)
  status?:      string | null          // null, "pending", "processing", "completed", "error"
  scenes:       SceneInfo[]            // empty = no scenes detected or not analyzed
  highlights:   HighlightInfo[]        // empty = no motion detected or not analyzed
}

SceneInfo {
  startTime:       float               // seconds (0.0 to duration)
  endTime:         float               // seconds (> startTime)
  changeIntensity: float               // 0.0 to 1.0 (higher = more dramatic)
}

HighlightInfo {
  timestamp: float                     // seconds (0.0 to duration)
  intensity: float                     // 0.0 to 1.0 (higher = more motion)
  type:      string                    // "motion"
}
```

---

## Analysis Status Flow

```
No Analysis Record
        ↓
   null status, null analysisId
        ↓
  [Trigger Analysis or Wait for Auto-Trigger]
        ↓
  "pending" status
        ↓
  [Queued, waiting for job processor]
        ↓
  "processing" status
        ↓
  [Analyzing preview frames...]
        ↓
  "completed" status
        ↓
  [Results available in scenes/highlights arrays]

Alternative path:
  "error" status → Analysis failed
```

---

## Response Examples

### Example 1: Analysis Not Yet Performed

```json
{
  "analysisId": null,
  "recordingId": 42,
  "status": null,
  "scenes": [],
  "highlights": []
}
```

**Meaning:**
- No analysis has been done for this recording
- Call POST endpoint to trigger analysis
- All fields except recordingId are null/empty

---

### Example 2: Analysis In Progress

```json
{
  "analysisId": null,
  "recordingId": 42,
  "status": "processing",
  "scenes": [],
  "highlights": []
}
```

**Meaning:**
- Analysis is currently running in background
- Results not yet available
- Retry this endpoint after a delay to get results

---

### Example 3: Analysis Completed with Scenes

```json
{
  "analysisId": 12,
  "recordingId": 42,
  "status": "completed",
  "scenes": [
    {
      "startTime": 0.0,
      "endTime": 25.3,
      "changeIntensity": 0.72
    },
    {
      "startTime": 25.3,
      "endTime": 60.8,
      "changeIntensity": 0.88
    },
    {
      "startTime": 60.8,
      "endTime": 120.0,
      "changeIntensity": 0.65
    }
  ],
  "highlights": [
    {
      "timestamp": 5.2,
      "intensity": 0.58,
      "type": "motion"
    },
    {
      "timestamp": 32.1,
      "intensity": 0.81,
      "type": "motion"
    },
    {
      "timestamp": 45.7,
      "intensity": 0.44,
      "type": "motion"
    },
    {
      "timestamp": 89.3,
      "intensity": 0.72,
      "type": "motion"
    }
  ]
}
```

**Meaning:**
- Analysis completed successfully
- Video has 3 scenes with intensity values ranging from 0.65 to 0.88
- 4 highlights detected with intensity varying from 0.44 to 0.81

---

### Example 4: Analysis Completed with No Highlights

```json
{
  "analysisId": 13,
  "recordingId": 43,
  "status": "completed",
  "scenes": [
    {
      "startTime": 0.0,
      "endTime": 180.0,
      "changeIntensity": 0.0
    }
  ],
  "highlights": []
}
```

**Meaning:**
- Video is one continuous scene (no scene changes detected)
- No significant motion detected (below 0.05 threshold)
- This is typical for static camera recordings

---

## Threshold Values

### Scene Detection

**SSIM Threshold:** 0.7

- Scenes are detected when SSIM (Structural Similarity Index) between consecutive frames drops below 0.7
- This means frames must be perceptually different by at least 30%
- Lower threshold = more sensitive to scene changes
- Higher threshold = only detects dramatic changes

---

### Highlight Detection

**Motion Threshold:** 0.05 (5%)

- Motion is recorded only when frame difference magnitude >= 0.05
- This is the normalized pixel-level difference threshold
- Lower threshold = more sensitive to small movements
- Higher threshold = only detects significant motion

---

## Data Characteristics

### Typical Scene Count

- **Static Camera (no cuts):** 1 scene
- **Interview/Presentation:** 2-5 scenes
- **Edited Video with Cuts:** 5-20 scenes
- **Fast-cut Video:** 20+ scenes

### Typical Highlight Count

- **Low Motion Content:** 0-5 highlights
- **Normal Content:** 5-15 highlights
- **High Motion/Dynamic Content:** 15+ highlights

### Typical Intensity Distribution

**Scene Change Intensity:**
- Average: 0.70-0.80 for typical cuts
- Range: 0.0 (no change) to 1.0 (complete difference)

**Highlight Intensity:**
- Average: 0.40-0.60 for detected motion
- Range: 0.05 (minimum threshold) to 1.0 (maximum motion)

---

## Notes

- All timestamps are in **seconds** with floating-point precision
- Analysis is **automatic** after preview frame generation
- Results are **persisted** in the database once analysis completes
- Status remains `"completed"` even after analysis is done; no need to re-analyze
- Empty arrays indicate either no analysis or no detectable scenes/highlights in that category
