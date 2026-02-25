---
title: ActivityPub
---

## Extensions

Workout Tracker exposes a small ActivityPub JSON-LD extension for workout outbox entries.

### Namespace

- Prefix: `aepy`
- Base URL: `http://joinaepyornis.orh/ns#`

### Terms

#### `aepy:workoutFitFile`

- Compact term: `workoutFitFile`
- IRI: `http://joinaepyornis.orh/ns#workoutFitFile`
- Used on: ActivityPub `Note` objects in workout outbox `Create` activities
- Value type: IRI (URL)
- Meaning: URL to the workout FIT file export

#### `aepy:workoutLocation`

- Compact term: `workoutLocation`
- Value type: string
- Meaning: Human readable workout location

#### `aepy:workoutSport`

- Compact term: `workoutSport`
- Value type: string
- Meaning: Workout sport or custom workout type

#### `aepy:workoutDuration`

- Compact term: `workoutDuration`
- Value type: integer
- Unit: seconds

#### Key metrics

The following compact terms are available as numeric metrics on workout `Note` objects:

- `workoutPauseDuration` (seconds)
- `workoutDistance` (meters)
- `workoutDistance2D` (meters)
- `workoutElevationGain` (meters)
- `workoutElevationLoss` (meters)
- `workoutAverageSpeed` (m/s)
- `workoutAverageSpeedMoving` (m/s)
- `workoutMaxSpeed` (m/s)
- `workoutAverageCadence`
- `workoutMaxCadence`
- `workoutAverageHeartRate`
- `workoutMaxHeartRate`
- `workoutAveragePower`
- `workoutMaxPower`
- `workoutRepetitions`
- `workoutWeight`

### Context fragment

```json
{
  "@context": [
    "https://www.w3.org/ns/activitystreams",
    {
      "aepy": "http://joinaepyornis.orh/ns#",
      "workoutFitFile": "aepy:workoutFitFile",
      "workoutLocation": "aepy:workoutLocation",
      "workoutSport": "aepy:workoutSport",
      "workoutDuration": "aepy:workoutDuration",
      "workoutPauseDuration": "aepy:workoutPauseDuration",
      "workoutDistance": "aepy:workoutDistance",
      "workoutDistance2D": "aepy:workoutDistance2D",
      "workoutElevationGain": "aepy:workoutElevationGain",
      "workoutElevationLoss": "aepy:workoutElevationLoss",
      "workoutAverageSpeed": "aepy:workoutAverageSpeed",
      "workoutAverageSpeedMoving": "aepy:workoutAverageSpeedMoving",
      "workoutMaxSpeed": "aepy:workoutMaxSpeed",
      "workoutAverageCadence": "aepy:workoutAverageCadence",
      "workoutMaxCadence": "aepy:workoutMaxCadence",
      "workoutAverageHeartRate": "aepy:workoutAverageHeartRate",
      "workoutMaxHeartRate": "aepy:workoutMaxHeartRate",
      "workoutAveragePower": "aepy:workoutAveragePower",
      "workoutMaxPower": "aepy:workoutMaxPower",
      "workoutRepetitions": "aepy:workoutRepetitions",
      "workoutWeight": "aepy:workoutWeight"
    }
  ]
}
```

### Example object fragment

```json
{
  "type": "Note",
  "workoutFitFile": "https://example.org/ap/users/alice/outbox/uuid/fit",
  "workoutLocation": "Brussels, Belgium",
  "workoutSport": "running",
  "workoutDuration": 3600,
  "workoutDistance": 10420,
  "workoutAverageSpeed": 2.89
}
```
