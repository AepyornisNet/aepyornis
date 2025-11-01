import {
  AfterViewInit,
  ChangeDetectionStrategy,
  Component,
  effect,
  ElementRef,
  inject,
  input,
  OnDestroy,
  viewChild,
  ViewEncapsulation,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import * as L from 'leaflet';
import { MapCenter, MapDataDetails } from '../../../../core/types/workout';
import { WorkoutDetailCoordinatorService } from '../../services/workout-detail-coordinator.service';
import { User } from '../../../../core/services/user';

type PolyLineProps = {
  renderer: L.Canvas;
  weight: number;
  interactive: boolean;
  color?: string;
};

@Component({
  selector: 'app-workout-map',
  imports: [CommonModule],
  encapsulation: ViewEncapsulation.None,
  templateUrl: './workout-map.html',
  styleUrls: ['./workout-map.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutMapComponent implements AfterViewInit, OnDestroy {
  private readonly mapContainer = viewChild<ElementRef<HTMLDivElement>>('mapContainer');

  public readonly mapData = input<MapDataDetails | undefined>();
  public readonly center = input<MapCenter | undefined>();

  private get mapDataValue(): MapDataDetails | undefined {
    return this.mapData();
  }

  private get centerValue(): MapCenter | undefined {
    return this.center();
  }

  private coordinatorService = inject(WorkoutDetailCoordinatorService);
  private userService = inject(User);
  private map?: L.Map;
  private trackGroup?: L.FeatureGroup;
  private hoverMarker?: L.CircleMarker;
  private highlightLayer?: L.FeatureGroup;
  private minElevation = 0;
  private maxElevation = 0;
  private maxSpeed = 0;

  public constructor() {
    // React to interval selection changes from the coordinator service
    effect(() => {
      const selection = this.coordinatorService.selectedInterval();
      this.highlightInterval(selection);
    });

    // Initialize map once inputs and view are ready
    effect(() => {
      const mapData = this.mapDataValue;
      const containerRef = this.mapContainer();
      if (!this.map && containerRef && mapData && mapData.position.length > 0) {
        this.initMap();
      }
    });
  }

  public ngAfterViewInit(): void {
    setTimeout(() => {
      this.initMap();
    }, 100);
  }

  public ngOnDestroy(): void {
    if (this.map) {
      this.map.remove();
    }
  }

  private initMap(): void {
    const containerRef = this.mapContainer();
    const mapData = this.mapDataValue;

    if (!containerRef || !mapData || mapData.position.length === 0) {
      return;
    }

    // Calculate min/max for color coding
    this.calculateMinMax();

    // Calculate center from points if not provided
    const midIndex = Math.floor(mapData.position.length / 2);
    const centerLat = this.centerValue?.lat ?? mapData.position[midIndex][0];
    const centerLng = this.centerValue?.lng ?? mapData.position[midIndex][1];

    // Initialize map
    this.map = L.map(containerRef.nativeElement, {
      fadeAnimation: false,
    }).setView([centerLat, centerLng], 15);

    // Add tile layers
    const streetLayer = L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '&copy; <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a>',
      className: 'map-tiles',
    });

    const aerialLayer = L.tileLayer(
      'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',
      {
        attribution: 'Powered by Esri',
      },
    );

    // Add default layer
    streetLayer.addTo(this.map);

    // Initialize track group
    this.trackGroup = L.featureGroup();

    // Create track renderer for performance
    const trackRenderer = L.canvas({ padding: 0.4 });
    const polyLineProperties = {
      renderer: trackRenderer,
      weight: 4,
      interactive: false,
    };

    // Draw the track with elevation coloring and add overlays
    const elevationLayer = this.drawElevationTrack(polyLineProperties);
    const speedLayer = this.drawSpeedTrack(polyLineProperties);
    const slopeLayer = this.drawSlopeTrack(polyLineProperties);

    // Add elevation layer by default
    if (elevationLayer) {
      elevationLayer.addTo(this.map);
    }

    // Add start and end markers
    this.addMarkers();

    // Add hover marker
    const firstPos = mapData.position[0];
    this.hoverMarker = L.circleMarker([firstPos[0], firstPos[1]], {
      color: 'blue',
      radius: 8,
    }).addTo(this.map);

    // Build overlay layers for control
    const overlays: Record<string, L.LayerGroup> = {
      Elevation: elevationLayer,
    };
    if (speedLayer) {
      overlays['Speed'] = speedLayer;
    }
    if (slopeLayer) {
      overlays['Slope'] = slopeLayer;
    }

    // Add scale control
    L.control.scale().addTo(this.map);

    // Add layer control
    L.control
      .layers(
        {
          Streets: streetLayer,
          Aerial: aerialLayer,
        },
        overlays,
        { position: 'topright' },
      )
      .addTo(this.map);

    // Fit bounds to track
    this.resetZoom();
  }

  private calculateMinMax(): void {
    const mapData = this.mapDataValue;
    if (!mapData) {
      return;
    }

    // Calculate min/max elevation
    const elevations = mapData.elevation.filter(
      (value): value is number => value !== null && value !== undefined,
    );
    if (elevations.length > 0) {
      this.minElevation = Math.min(...elevations);
      this.maxElevation = Math.max(...elevations);
    } else {
      this.minElevation = 0;
      this.maxElevation = 0;
    }

    // Calculate max speed
    const speeds = mapData.speed.filter(
      (value): value is number => value !== null && value !== undefined && value > 0,
    );
    this.maxSpeed = speeds.length > 0 ? Math.max(...speeds) : 0;
  }

  private drawElevationTrack(polyLineProperties: PolyLineProps): L.FeatureGroup {
    const elevationLayer = L.featureGroup();

    const mapData = this.mapDataValue;
    if (!this.map || !mapData || !this.trackGroup) {
      return elevationLayer;
    }

    const trackRenderer = polyLineProperties.renderer;
    let prevPoint: [number, number] | null = null;

    mapData.position.forEach((pos, index) => {
      if (prevPoint) {
        const elevation = mapData.elevation[index] ?? 0;

        // Add invisible point for tooltip
        this.trackGroup!.addLayer(
          L.circleMarker([pos[0], pos[1]], {
            renderer: trackRenderer,
            opacity: 0,
            fill: false,
            radius: 4,
          })
            .addTo(this.map!)
            .bindTooltip(() => this.getTooltip(index)),
        );

        // Color based on elevation
        const color = this.getColor(
          (elevation - this.minElevation) / (this.maxElevation - this.minElevation),
        );

        L.polyline([prevPoint, [pos[0], pos[1]]], {
          ...polyLineProperties,
          color,
        }).addTo(elevationLayer);
      }
      prevPoint = [pos[0], pos[1]];
    });

    return elevationLayer;
  }

  private drawSpeedTrack(polyLineProperties: PolyLineProps): L.FeatureGroup | null {
    const mapData = this.mapDataValue;
    if (!mapData?.speed || mapData.speed.length === 0) {
      return null;
    }

    const speedLayer = L.featureGroup();
    const MOVING_AVERAGE_LENGTH = 15;
    const movingSpeeds: number[] = [];

    // Calculate average and standard deviation
    const speeds = mapData.speed.filter(
      (value): value is number => value !== null && value !== undefined && value > 0,
    );
    const averageSpeed = speeds.reduce((a, x) => a + x, 0) / speeds.length;
    const stdevSpeed = Math.sqrt(
      speeds.reduce((a, x) => a + Math.pow(x - averageSpeed, 2), 0) / (speeds.length - 1),
    );

    let prevPoint: [number, number] | null = null;

    mapData.position.forEach((pos, index) => {
      if (prevPoint) {
        const speed = mapData.speed[index];
        let color: string;

        if (speed === null || speed === undefined || speed < 0.1) {
          color = 'rgb(0,0,0)'; // Pausing
        } else {
          if (movingSpeeds.length > MOVING_AVERAGE_LENGTH) {
            movingSpeeds.shift();
          }
          movingSpeeds.push(speed);
          const movingAverageSpeed = movingSpeeds.reduce((a, x) => a + x) / movingSpeeds.length;

          const zScore = ((movingAverageSpeed || averageSpeed) - averageSpeed) / stdevSpeed;
          color = this.getColor(0.5 + zScore / 2);
        }

        L.polyline([prevPoint, [pos[0], pos[1]]], {
          ...polyLineProperties,
          color,
        }).addTo(speedLayer);
      }
      prevPoint = [pos[0], pos[1]];
    });

    return speedLayer;
  }

  private drawSlopeTrack(polyLineProperties: PolyLineProps): L.FeatureGroup | null {
    const mapData = this.mapDataValue;
    if (!mapData?.slope || mapData.slope.length === 0) {
      return null;
    }

    const slopeLayer = L.featureGroup();

    const slopes = mapData.slope.filter(
      (value): value is number => value !== null && value !== undefined,
    );
    const maxSlope = Math.max(...slopes);
    const minSlope = Math.min(...slopes);

    let prevPoint: [number, number] | null = null;

    mapData.position.forEach((pos, index) => {
      if (prevPoint) {
        const slope = mapData.slope[index] ?? 0;
        const zScore = (slope - minSlope) / (maxSlope - minSlope);
        const color = this.getColor(zScore);

        L.polyline([prevPoint, [pos[0], pos[1]]], {
          ...polyLineProperties,
          color,
        }).addTo(slopeLayer);
      }
      prevPoint = [pos[0], pos[1]];
    });

    return slopeLayer;
  }

  private addMarkers(): void {
    const mapData = this.mapDataValue;
    if (!this.map || !mapData || !this.trackGroup) {
      return;
    }

    const positions = mapData.position;

    // Add end marker (red)
    const lastPos = positions[positions.length - 1];
    this.trackGroup.addLayer(
      L.circleMarker([lastPos[0], lastPos[1]], {
        color: 'red',
        fill: true,
        fillColor: 'red',
        fillOpacity: 1,
        radius: 6,
      })
        .addTo(this.map)
        .bindTooltip(this.getTooltip(positions.length - 1)),
    );

    // Add start marker (green)
    const firstPos = positions[0];
    this.trackGroup.addLayer(
      L.circleMarker([firstPos[0], firstPos[1]], {
        color: 'green',
        fill: true,
        fillColor: 'green',
        fillOpacity: 1,
        radius: 6,
      })
        .addTo(this.map)
        .bindTooltip(this.getTooltip(0)),
    );
  }

  private getTooltip(index: number): string {
    const mapData = this.mapDataValue;
    if (!mapData) {
      return '';
    }

    const userInfo = this.userService.getUserInfo()();
    const distanceUnit = userInfo?.profile?.preferred_units?.distance || 'km';
    const speedUnit = userInfo?.profile?.preferred_units?.speed || 'km/h';
    const elevationUnit = userInfo?.profile?.preferred_units?.elevation || 'm';

    let tooltip = '<ul style="list-style: none; padding: 0; margin: 0;">';

    // Time
    if (mapData.time[index]) {
      const time = new Date(mapData.time[index]).toTimeString().substring(0, 5);
      tooltip += `<li><b>Time</b>: ${time}</li>`;
    }

    // Distance
    if (mapData.distance[index] !== undefined) {
      const distanceKm = mapData.distance[index];
      let displayDistance: number;
      if (distanceUnit === 'mi') {
        displayDistance = distanceKm * 0.621371;
      } else {
        displayDistance = distanceKm;
      }
      tooltip += `<li><b>Distance</b>: ${displayDistance.toFixed(2)} ${distanceUnit}</li>`;
    }

    // Duration
    if (mapData.duration[index] !== undefined) {
      tooltip += `<li><b>Duration</b>: ${this.formatDuration(mapData.duration[index])}</li>`;
    }

    // Speed
    if (mapData.speed[index] !== undefined && mapData.speed[index] !== null) {
      const speedMps = mapData.speed[index] as number;
      let displaySpeed: number;
      if (speedUnit === 'mph') {
        displaySpeed = speedMps * 2.23694;
      } else {
        displaySpeed = speedMps * 3.6;
      }
      tooltip += `<li><b>Speed</b>: ${displaySpeed.toFixed(2)} ${speedUnit}</li>`;
    }

    // Elevation
    if (mapData.elevation[index] !== undefined) {
      const elevationM = mapData.elevation[index] as number;
      let displayElevation: number;
      if (elevationUnit === 'ft') {
        displayElevation = elevationM * 3.28084;
      } else {
        displayElevation = elevationM;
      }
      tooltip += `<li><b>Elevation</b>: ${displayElevation.toFixed(1)} ${elevationUnit}</li>`;
    }

    // Slope
    if (mapData.slope && mapData.slope[index] !== undefined && mapData.slope[index] !== null) {
      tooltip += `<li><b>Slope</b>: ${mapData.slope[index]!.toFixed(1)}%</li>`;
    }

    // Extra metrics
    if (mapData.extra_metrics) {
      if (mapData.extra_metrics['heart-rate']?.[index]) {
        tooltip += `<li><b>Heart Rate</b>: ${Math.round(mapData.extra_metrics['heart-rate'][index] as number)} bpm</li>`;
      }
      if (mapData.extra_metrics['cadence']?.[index]) {
        tooltip += `<li><b>Cadence</b>: ${Math.round(mapData.extra_metrics['cadence'][index] as number)}</li>`;
      }
      if (mapData.extra_metrics['temperature']?.[index]) {
        tooltip += `<li><b>Temperature</b>: ${(mapData.extra_metrics['temperature'][index] as number).toFixed(1)} °C</li>`;
      }
    }

    tooltip += '</ul>';
    return tooltip;
  }

  private formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = Math.floor(seconds % 60);

    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }
    return `${minutes}:${secs.toString().padStart(2, '0')}`;
  }

  private getColor(value: number): string {
    // Clamp value between 0 and 1
    value = Math.max(0, Math.min(1, value));

    // Interpolate between blue and green
    const lowColor = [50, 50, 255];
    const highColor = [50, 255, 50];
    const color = [0, 1, 2].map((i) =>
      Math.floor(value * (highColor[i] - lowColor[i]) + lowColor[i]),
    );

    return `rgb(${color.join(',')})`;
  }

  private highlightInterval(selection: { startIndex: number; endIndex: number } | null): void {
    // Remove existing highlight
    if (this.highlightLayer) {
      this.highlightLayer.clearLayers();
    }

    const mapData = this.mapDataValue;
    if (!selection || !this.map || !mapData) {
      // If clearing, also reset zoom to show full track
      if (!selection && this.map) {
        this.resetZoom();
      }
      return;
    }

    // Create or reuse highlight layer
    if (!this.highlightLayer) {
      this.highlightLayer = L.featureGroup().addTo(this.map);
    }

    // Draw highlighted segment using red color like the original map.js
    const positions = mapData.position.slice(selection.startIndex, selection.endIndex + 1);

    for (let i = 1; i < positions.length; i++) {
      const prevPos: L.LatLngExpression = [positions[i - 1][0], positions[i - 1][1]];
      const currPos: L.LatLngExpression = [positions[i][0], positions[i][1]];

      L.polyline([prevPos, currPos], {
        color: 'red',
        weight: 5,
        opacity: 0.8,
      }).addTo(this.highlightLayer);
    }
  }

  private resetZoom(): void {
    if (this.map && this.trackGroup) {
      this.map.fitBounds(this.trackGroup.getBounds(), { animate: false });
    }
  }
}
