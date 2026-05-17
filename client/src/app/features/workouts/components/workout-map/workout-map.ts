import {
  ApplicationRef,
  ChangeDetectionStrategy,
  Component,
  computed,
  createComponent,
  effect,
  EnvironmentInjector,
  inject,
  input,
  OnDestroy,
  signal,
} from '@angular/core';

import { NgxMapLibreGLModule } from '@maplibre/ngx-maplibre-gl';
import maplibregl, { LngLatBounds, Map } from 'maplibre-gl';
import { MapCenter, MapDataDetails } from '../../../../core/types/workout';
import { WorkoutDetailCoordinatorService } from '../../services/workout-detail-coordinator.service';
import { User } from '../../../../core/services/user';
import {
  DEFAULT_HEART_RATE_COLOR,
  DEFAULT_POWER_COLOR,
  FTP_ZONE_COLORS,
  HR_ZONE_COLORS,
} from '../zone-colors';
import { BaseMapComponent } from '../../../../core/components/base-map/base-map';
import {
  faSolidBolt,
  faSolidGaugeHigh,
  faSolidHeart,
  faSolidMountain,
  faSolidXmark,
} from '@ng-icons/font-awesome/solid';
import { _ } from '@ngx-translate/core';
import {
  WorkoutMapPointPopupComponent,
  WorkoutMapPointPopupData,
} from '../workout-map-point-popup/workout-map-point-popup';

type MetricLayer = 'none' | 'elevation' | 'power' | 'speed' | 'heartrate';

@Component({
  selector: 'app-workout-map',
  imports: [NgxMapLibreGLModule],
  templateUrl: './workout-map.html',
  styleUrls: ['./workout-map.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutMapComponent extends BaseMapComponent implements OnDestroy {
  public readonly mapData = input<MapDataDetails | undefined>();
  public readonly center = input<MapCenter | undefined>();

  private readonly coordinatorService = inject(WorkoutDetailCoordinatorService);
  private readonly userService = inject(User);
  private readonly environmentInjector = inject(EnvironmentInjector);
  private readonly applicationRef = inject(ApplicationRef);

  private startMarker?: maplibregl.Marker;
  private endMarker?: maplibregl.Marker;
  private hoverPopup?: maplibregl.Popup;
  private hoverPopupComponentRef?: ReturnType<
    typeof createComponent<WorkoutMapPointPopupComponent>
  >;
  private metricControl?: maplibregl.IControl;
  private readonly metricButtons: Partial<Record<MetricLayer, HTMLButtonElement>> = {};

  private minElevation = 0;
  private maxElevation = 0;

  public readonly filteredMapData = computed<MapDataDetails | undefined>(() => {
    const mapData = structuredClone(this.mapData());
    if (!mapData) {
      return;
    }

    mapData.position = mapData.position.filter((p, idx) => {
      if (p[0] !== 0 || p[1] !== 0) {
        return true;
      }

      if (mapData.time) {
        mapData.time.splice(idx, 1);
      }

      if (mapData.distance) {
        mapData.distance.splice(idx, 1);
      }

      if (mapData.duration) {
        mapData.duration.splice(idx, 1);
      }

      if (mapData.elevation) {
        mapData.elevation.splice(idx, 1);
      }

      if (mapData.slope) {
        mapData.slope.splice(idx, 1);
      }

      if (mapData.speed) {
        mapData.speed.splice(idx, 1);
      }

      if (mapData.extra_metrics?.['power']) {
        mapData.extra_metrics['power'].splice(idx, 1);
      }

      if (mapData.extra_metrics?.['heart-rate']) {
        mapData.extra_metrics['heart-rate'].splice(idx, 1);
      }

      return false;
    });

    return mapData;
  });

  public readonly activeMetric = signal<MetricLayer>('elevation');
  public readonly availableMetrics = computed<MetricLayer[]>(() => {
    const data = this.filteredMapData();
    if (!data) {
      return ['none'];
    }

    const metrics: MetricLayer[] = ['none'];
    if (this.hasNumericData(data.elevation)) {
      metrics.push('elevation');
    }
    if (this.hasNumericData(data.extra_metrics?.['power'])) {
      metrics.push('power');
    }
    if (this.hasNumericData(data.speed)) {
      metrics.push('speed');
    }
    if (this.hasNumericData(data.extra_metrics?.['heart-rate'])) {
      metrics.push('heartrate');
    }
    return metrics;
  });

  public readonly initialCenter = computed<[number, number]>(() => {
    const fromInput = this.center();
    if (fromInput) {
      return [fromInput.lng, fromInput.lat];
    }

    const data = this.filteredMapData();
    if (!data?.position?.length) {
      return [0, 0];
    }

    const mid = Math.floor(data.position.length / 2);
    return [data.position[mid][1], data.position[mid][0]];
  });

  public constructor() {
    super();
    effect(() => {
      this.filteredMapData();
      const available = this.availableMetrics();
      if (!available.includes(this.activeMetric())) {
        this.activeMetric.set('none');
      }
      if (this.map && this.map.isStyleLoaded()) {
        this.renderTrackLayers();
      }
    });

    effect(() => {
      this.applyMetricVisibility(this.activeMetric());
    });

    effect(() => {
      this.highlightInterval(this.coordinatorService.selectedInterval());
    });

    effect(() => {
      this.highlightPoint(this.coordinatorService.lastHoveredIdx());
    });
  }

  public onMapLoad(map: Map): void {
    this.onMapLoadBase(map);
    this.refreshTrackAfterStyleChange();
  }

  public override ngOnDestroy(): void {
    this.destroyHoverPopupComponent();
    super.ngOnDestroy();
  }

  public setMetric(metric: MetricLayer): void {
    if (!this.availableMetrics().includes(metric)) {
      return;
    }
    this.activeMetric.set(metric);
    this.updateMetricControlState();
  }

  private renderTrackLayers(): void {
    const mapData = this.filteredMapData();
    if (!this.map || !mapData || mapData.position.length < 2) {
      return;
    }

    this.calculateMinMax(mapData);
    this.clearLayers();
    this.addOverviewLayer(mapData);

    this.addMetricLayer('track-elevation-source', 'track-elevation-layer', mapData, (index) => {
      const elevation = mapData.elevation[index] ?? 0;
      const normalized =
        (elevation - this.minElevation) / Math.max(this.maxElevation - this.minElevation, 1);
      return this.getColor(normalized);
    });

    const speeds = mapData.speed.filter(
      (value): value is number => value !== null && value !== undefined && value > 0,
    );
    const average = speeds.length ? speeds.reduce((a, b) => a + b, 0) / speeds.length : 0;
    const stdev =
      speeds.length > 1
        ? Math.sqrt(speeds.reduce((a, x) => a + Math.pow(x - average, 2), 0) / (speeds.length - 1))
        : 1;
    this.addMetricLayer('track-speed-source', 'track-speed-layer', mapData, (index) => {
      const speed = mapData.speed[index] ?? 0;
      const zScore = (speed - average) / (stdev || 1);
      return this.getColor(0.5 + zScore / 2);
    });

    if (this.hasNumericData(mapData.extra_metrics?.['power'])) {
      this.addMetricLayer('track-power-source', 'track-power-layer', mapData, (index) => {
        const value = mapData.extra_metrics?.['power']?.[index];
        return this.getZoneColor('power', value);
      });
    }

    if (this.hasNumericData(mapData.extra_metrics?.['heart-rate'])) {
      this.addMetricLayer('track-heartrate-source', 'track-heartrate-layer', mapData, (index) => {
        const value = mapData.extra_metrics?.['heart-rate']?.[index];
        return this.getZoneColor('heart-rate', value);
      });
    }

    const tooltipFeatures = mapData.position.map((pos, index) => ({
      type: 'Feature' as const,
      geometry: { type: 'Point' as const, coordinates: [pos[1], pos[0]] },
      properties: { index },
    }));

    this.map.addSource('track-points', {
      type: 'geojson',
      data: {
        type: 'FeatureCollection',
        features: tooltipFeatures,
      },
    });

    this.map.addLayer({
      id: 'track-points-layer',
      type: 'circle',
      source: 'track-points',
      paint: {
        'circle-radius': 6,
        'circle-opacity': 0,
      },
    });

    this.map.on('mousemove', 'track-points-layer', (event) => {
      const feature = event.features?.[0];
      const index = Number(feature?.properties?.['index']);
      if (!Number.isFinite(index) || !event.lngLat) {
        return;
      }

      if (!this.hoverPopup) {
        this.hoverPopup = new maplibregl.Popup({
          closeButton: false,
          closeOnClick: false,
          offset: 10,
        });
      }

      const popupData = this.getTooltipData(index);
      if (!popupData) {
        return;
      }

      const componentRef = createComponent(WorkoutMapPointPopupComponent, {
        environmentInjector: this.environmentInjector,
      });
      componentRef.setInput('data', popupData);
      this.applicationRef.attachView(componentRef.hostView);
      this.destroyHoverPopupComponent();
      this.hoverPopupComponentRef = componentRef;

      const popupElement = componentRef.location.nativeElement as HTMLElement;
      this.hoverPopup.setLngLat(event.lngLat).setDOMContent(popupElement).addTo(this.map!);
    });

    this.map.on('mouseleave', 'track-points-layer', () => {
      this.hoverPopup?.remove();
      this.destroyHoverPopupComponent();
    });

    const firstPos = mapData.position[0];
    this.startMarker = new maplibregl.Marker({ color: 'green', scale: 0.8 })
      .setLngLat([firstPos[1], firstPos[0]])
      .addTo(this.map);

    const lastPos = mapData.position[mapData.position.length - 1];
    this.endMarker = new maplibregl.Marker({ color: 'red', scale: 0.8 })
      .setLngLat([lastPos[1], lastPos[0]])
      .addTo(this.map);

    this.applyMetricVisibility(this.activeMetric());
    this.resetZoom();
  }

  private addMetricLayer(
    sourceId: string,
    layerId: string,
    mapData: MapDataDetails,
    colorResolver: (index: number) => string,
  ): void {
    if (!this.map) {
      return;
    }

    const features = [] as GeoJSON.Feature<GeoJSON.LineString, { color: string }>[];

    for (let i = 1; i < mapData.position.length; i++) {
      const prev = mapData.position[i - 1];
      const curr = mapData.position[i];
      features.push({
        type: 'Feature',
        geometry: {
          type: 'LineString',
          coordinates: [
            [prev[1], prev[0]],
            [curr[1], curr[0]],
          ],
        },
        properties: {
          color: colorResolver(i),
        },
      });
    }

    this.map.addSource(sourceId, {
      type: 'geojson',
      tolerance: 0,
      data: {
        type: 'FeatureCollection',
        features,
      },
    });

    this.map.addLayer({
      id: layerId,
      type: 'line',
      source: sourceId,
      paint: {
        'line-color': ['get', 'color'],
        'line-width': ['interpolate', ['linear'], ['zoom'], 0, 2, 10, 3, 15, 4],
      },
      layout: {
        visibility: 'none',
        'line-cap': 'round',
        'line-join': 'round',
      },
    });
  }

  private addOverviewLayer(mapData: MapDataDetails): void {
    if (!this.map) {
      return;
    }

    const coordinates = mapData.position.map(
      (position) => [position[1], position[0]] as [number, number],
    );

    this.map.addSource('track-overview-source', {
      type: 'geojson',
      data: {
        type: 'FeatureCollection',
        features: [
          {
            type: 'Feature',
            geometry: { type: 'LineString', coordinates },
            properties: {},
          },
        ],
      },
    });

    this.map.addLayer({
      id: 'track-overview-layer',
      type: 'line',
      source: 'track-overview-source',
      paint: {
        'line-color': '#16a34a',
        'line-width': ['interpolate', ['linear'], ['zoom'], 0, 2, 10, 3, 15, 5],
        'line-opacity': 0.7,
      },
    });
  }

  private highlightInterval(selection: { startIndex: number; endIndex: number } | null): void {
    if (!this.map || !this.filteredMapData()) {
      return;
    }

    if (this.map.getLayer('highlight-layer')) {
      this.map.removeLayer('highlight-layer');
    }
    if (this.map.getSource('highlight-source')) {
      this.map.removeSource('highlight-source');
    }

    if (!selection) {
      this.resetZoom();
      return;
    }

    const positions = this.filteredMapData()!.position.slice(
      selection.startIndex,
      selection.endIndex + 1,
    );
    const coordinates = positions.map((position) => [position[1], position[0]] as [number, number]);

    this.map.addSource('highlight-source', {
      type: 'geojson',
      data: {
        type: 'FeatureCollection',
        features: [
          {
            type: 'Feature',
            geometry: { type: 'LineString', coordinates },
            properties: {},
          },
        ],
      },
    });

    this.map.addLayer({
      id: 'highlight-layer',
      type: 'line',
      source: 'highlight-source',
      paint: {
        'line-color': 'red',
        'line-width': 5,
        'line-opacity': 0.8,
      },
    });
  }

  private highlightPoint(selection: number | null): void {
    if (!this.map || !this.filteredMapData()) {
      return;
    }

    if (this.map.getLayer('hover-highlight-layer')) {
      this.map.removeLayer('hover-highlight-layer');
    }
    if (this.map.getSource('hover-highlight-source')) {
      this.map.removeSource('hover-highlight-source');
    }

    if (selection === null) {
      return;
    }

    const p = this.filteredMapData()!.position[selection];
    this.map.addSource('hover-highlight-source', {
      type: 'geojson',
      data: {
        type: 'FeatureCollection',
        features: [
          {
            type: 'Feature',
            geometry: { type: 'Point', coordinates: [p[1], p[0]] },
            properties: null,
          },
        ],
      },
    });

    this.map.addLayer({
      id: 'hover-highlight-layer',
      type: 'circle',
      source: 'hover-highlight-source',
      paint: {
        'circle-radius': 5,
        'circle-opacity': 0,
        'circle-stroke-opacity': 1,
        'circle-stroke-color': '#a20000',
        'circle-stroke-width': 3,
      },
    });
  }

  private applyMetricVisibility(metric: MetricLayer): void {
    if (!this.map) {
      return;
    }

    const mapping: Record<MetricLayer, string> = {
      none: '',
      elevation: 'track-elevation-layer',
      power: 'track-power-layer',
      speed: 'track-speed-layer',
      heartrate: 'track-heartrate-layer',
    };

    for (const [key, layerId] of Object.entries(mapping) as [MetricLayer, string][]) {
      if (!layerId) {
        continue;
      }
      if (!this.map.getLayer(layerId)) {
        continue;
      }
      this.map.setLayoutProperty(layerId, 'visibility', key === metric ? 'visible' : 'none');
    }
    this.updateMetricControlState();
  }

  protected refreshAfterStyleChange(): void {
    this.refreshTrackAfterStyleChange();
  }

  private refreshTrackAfterStyleChange(): void {
    if (!this.map) {
      return;
    }

    const version = ++this.styleRefreshVersion;

    const tryRender = (): void => {
      if (!this.map || version !== this.styleRefreshVersion) {
        return;
      }
      if (this.map.isStyleLoaded()) {
        this.renderTrackLayers();
        return;
      }
      window.setTimeout(tryRender, 50);
    };

    window.setTimeout(tryRender, 0);
  }

  private calculateMinMax(mapData: MapDataDetails): void {
    const elevations = mapData.elevation.filter(
      (value): value is number => value !== null && value !== undefined,
    );
    this.minElevation = elevations.length ? Math.min(...elevations) : 0;
    this.maxElevation = elevations.length ? Math.max(...elevations) : 0;
  }

  private clearLayers(): void {
    if (!this.map) {
      return;
    }

    const layers = [
      'highlight-layer',
      'track-overview-layer',
      'track-points-layer',
      'track-elevation-layer',
      'track-power-layer',
      'track-speed-layer',
      'track-heartrate-layer',
    ];

    const sources = [
      'highlight-source',
      'track-overview-source',
      'track-points',
      'track-elevation-source',
      'track-power-source',
      'track-speed-source',
      'track-heartrate-source',
    ];

    for (const layer of layers) {
      if (this.map.getLayer(layer)) {
        this.map.removeLayer(layer);
      }
    }

    for (const source of sources) {
      if (this.map.getSource(source)) {
        this.map.removeSource(source);
      }
    }

    this.startMarker?.remove();
    this.endMarker?.remove();
    this.hoverPopup?.remove();
    this.destroyHoverPopupComponent();
  }

  private resetZoom(): void {
    const mapData = this.filteredMapData();
    if (!this.map || !mapData?.position?.length) {
      return;
    }

    const bounds = new LngLatBounds(
      [mapData.position[0][1], mapData.position[0][0]],
      [mapData.position[0][1], mapData.position[0][0]],
    );

    for (const point of mapData.position) {
      bounds.extend([point[1], point[0]]);
    }

    this.map.fitBounds(bounds, { padding: 60, animate: false });
  }

  private getTooltipData(index: number): WorkoutMapPointPopupData | null {
    const mapData = this.filteredMapData();
    if (!mapData) {
      return null;
    }

    const userInfo = this.userService.getUserInfo()();
    return {
      time: mapData.time[index],
      distance: mapData.distance[index],
      duration: mapData.duration[index],
      speed: mapData.speed[index],
      elevation: mapData.elevation[index],
      slope: mapData.slope?.[index],
      heartRate: mapData.extra_metrics?.['heart-rate']?.[index],
      power: mapData.extra_metrics?.['power']?.[index],
      distanceUnit: userInfo?.profile?.profile?.preferred_units?.distance || 'km',
      speedUnit: userInfo?.profile?.profile?.preferred_units?.speed || 'km/h',
      elevationUnit: userInfo?.profile?.profile?.preferred_units?.elevation || 'm',
    };
  }

  private destroyHoverPopupComponent(): void {
    if (!this.hoverPopupComponentRef) {
      return;
    }
    this.applicationRef.detachView(this.hoverPopupComponentRef.hostView);
    this.hoverPopupComponentRef.destroy();
    this.hoverPopupComponentRef = undefined;
  }

  private getColor(value: number): string {
    const normalized = Math.max(0, Math.min(1, value));
    const lowColor = [50, 50, 255];
    const highColor = [50, 255, 50];
    const color = [0, 1, 2].map((i) =>
      Math.floor(normalized * (highColor[i] - lowColor[i]) + lowColor[i]),
    );
    return `rgb(${color.join(',')})`;
  }

  protected override addMapControls(): void {
    super.addMapControls();

    if (!this.map || this.metricControl) {
      return;
    }

    this.metricControl = this.createMetricControl();
    this.map.addControl(this.metricControl, 'top-right');
    this.updateMetricControlState();
  }

  private createMetricControl(): maplibregl.IControl {
    const container = document.createElement('div');
    container.className = 'maplibregl-ctrl maplibregl-ctrl-group wt-map-control';

    const entries: { key: MetricLayer; label: string; title: string }[] = [
      { key: 'none', label: faSolidXmark, title: _('None') },
      { key: 'elevation', label: faSolidMountain, title: _('Elevation') },
      { key: 'power', label: faSolidBolt, title: _('Power') },
      { key: 'speed', label: faSolidGaugeHigh, title: _('Speed') },
      { key: 'heartrate', label: faSolidHeart, title: _('Heart rate') },
    ];

    for (const entry of entries) {
      const button = this.createControlButton(entry.label, entry.title, () =>
        this.setMetric(entry.key),
      );
      button.classList.add(`wt-map-control-button--${entry.key}`);
      this.metricButtons[entry.key] = button;
      container.append(button);
    }

    return {
      onAdd: (): HTMLElement => container,
      onRemove: (): void => {
        container.remove();
      },
      getDefaultPosition: () => 'top-right' as const,
    };
  }

  private updateMetricControlState(): void {
    const activeMetric = this.activeMetric();
    const available = new Set(this.availableMetrics());

    for (const [metric, button] of Object.entries(this.metricButtons) as [
      MetricLayer,
      HTMLButtonElement | undefined,
    ][]) {
      if (!button) {
        continue;
      }
      const isAvailable = available.has(metric);
      button.style.display = isAvailable ? '' : 'none';
      button.classList.toggle('is-active', metric === activeMetric);
      button.setAttribute('aria-pressed', String(metric === activeMetric));
    }
  }

  private hasNumericData(values: (number | null | undefined)[] | undefined): boolean {
    if (!values || values.length === 0) {
      return false;
    }

    return values.some((value) => typeof value === 'number' && Number.isFinite(value));
  }

  private getZoneColor(type: 'heart-rate' | 'power', value: number | null | undefined): string {
    const palette = type === 'heart-rate' ? HR_ZONE_COLORS : FTP_ZONE_COLORS;
    const fallback = type === 'heart-rate' ? DEFAULT_HEART_RATE_COLOR : DEFAULT_POWER_COLOR;
    if (typeof value !== 'number' || !Number.isFinite(value)) {
      return fallback;
    }

    const zoneRanges = this.filteredMapData()?.zone_ranges?.[type];
    if (!zoneRanges || zoneRanges.length === 0) {
      return fallback;
    }

    for (const zone of zoneRanges) {
      const min = zone.min;
      const max = zone.max;
      const meetsMin = min === null || min === undefined || value >= min;
      const meetsMax = max === null || max === undefined || value <= max;
      if (meetsMin && meetsMax) {
        return palette[zone.zone] ?? fallback;
      }
    }

    return fallback;
  }
}
