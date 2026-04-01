import {
  ApplicationRef,
  ChangeDetectionStrategy,
  Component,
  createComponent,
  EnvironmentInjector,
  inject,
  signal,
} from '@angular/core';

import { FormsModule } from '@angular/forms';
import { TranslatePipe } from '@ngx-translate/core';
import { firstValueFrom } from 'rxjs';
import { NgxMapLibreGLModule } from '@maplibre/ngx-maplibre-gl';
import maplibregl, { LngLatBounds, Map } from 'maplibre-gl';
import type { ExpressionSpecification } from 'maplibre-gl';
import { Api } from '../../../../core/services/api';
import { WorkoutPopupData } from '../../../../core/types/statistics';
import { WorkoutPopup } from '../../components/workout-popup/workout-popup';
import { BaseMapComponent } from '../../../../core/components/base-map/base-map';

const DEFAULT_HEATMAP_CELL_SIZE = 0.0015;
const MEDIUM_HEATMAP_CELL_SIZE = 0.0007;
const FINE_HEATMAP_CELL_SIZE = 0.0003;
const HEAT_SOURCE_ID = 'heat-source';
const HEAT_LAYER_ID = 'heat-layer';
const HEAT_SOFTENER_LAYER_ID = 'heat-points-softener';
const MARKERS_SOURCE_ID = 'markers-source';
const CLUSTERS_LAYER_ID = 'clusters';
const CLUSTER_COUNT_LAYER_ID = 'cluster-count';
const UNCLUSTERED_LAYER_ID = 'unclustered-point';
const HEATMAP_COLOR_SCALE: ExpressionSpecification = [
  'interpolate',
  ['linear'],
  ['heatmap-density'],
  0,
  'rgba(0,0,255,0)',
  0.25,
  'rgb(0,0,255)',
  0.5,
  'rgb(0,255,0)',
  0.75,
  'rgb(255,255,0)',
  1,
  'rgb(255,0,0)',
];

@Component({
  selector: 'app-heatmap',
  imports: [FormsModule, TranslatePipe, NgxMapLibreGLModule],
  templateUrl: './heatmap.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  styleUrls: ['./heatmap.scss'],
})
export class Heatmap extends BaseMapComponent {
  private readonly api = inject(Api);
  private readonly environmentInjector = inject(EnvironmentInjector);
  private readonly applicationRef = inject(ApplicationRef);

  private markerPopup?: maplibregl.Popup;
  private coordinatesRefreshVersion = 0;
  private coordinatesRequestKey: string | null = null;
  private markerInteractionsBound = false;
  private maxPointCount = 1;

  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);

  public readonly radius = signal(10);
  public readonly blur = signal(15);
  public readonly showMarkers = signal(true);
  public readonly onlyTrace = signal(false);

  private heatMapData: [number, number, number][] = [];
  private markerFeatures: GeoJSON.Feature[] = [];

  protected override onDestroy(): void {
    this.map?.off('moveend', this.onMapMoveEnd);
  }

  public async onMapLoad(map: Map): Promise<void> {
    this.onMapLoadBase(map);
    this.bindMarkerInteractions();
    this.refreshHeatmapAfterStyleChange();
    await this.loadHeatmapData();
    this.map!.on('moveend', this.onMapMoveEnd);
  }

  public onRenderSettingsChange(): void {
    void this.refreshCoordinatesIfNeeded();
    this.updateHeatmapPaint();
    this.setMarkerVisibility(this.showMarkers());
  }

  private async loadHeatmapData(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const centersResponse = await firstValueFrom(this.api.getWorkoutsCenters());

      const centerGeoJson = (centersResponse?.results as GeoJSON.FeatureCollection | undefined) ?? {
        type: 'FeatureCollection',
        features: [],
      };
      this.markerFeatures = (centerGeoJson.features ?? []).map((feature) => {
        const props = (feature.properties ?? {}) as Record<string, unknown>;
        const popupData = props['popup_data'];
        return {
          ...feature,
          properties: {
            ...props,
            popup_data:
              popupData && typeof popupData !== 'string' ? JSON.stringify(popupData) : popupData,
          },
        };
      });

      this.fitInitialBounds();
      await this.refreshCoordinatesIfNeeded(true, false);
    } catch (err) {
      console.error('Failed to load heatmap data:', err);
      this.error.set('Failed to load heatmap data. Please try again.');
    } finally {
      this.loading.set(false);
    }
  }

  private rerenderHeatMap(): void {
    if (!this.map) {
      return;
    }

    this.upsertHeatSource();
    this.upsertHeatLayers();
    this.updateHeatmapPaint();
    this.upsertMarkersSource();
    this.upsertMarkerLayers();
    this.setMarkerVisibility(this.showMarkers());
  }

  private upsertHeatSource(): void {
    if (!this.map) {
      return;
    }

    const maxWeight = this.heatMapData.reduce((max, [, , weight]) => Math.max(max, weight ?? 1), 1);
    const features: GeoJSON.Feature<GeoJSON.Point, { weight: number }>[] = this.heatMapData.map(
      ([lat, lng, weight]) => ({
        type: 'Feature',
        geometry: {
          type: 'Point',
          coordinates: [lng, lat],
        },
        properties: { weight: Math.min(1, (weight ?? 1) / maxWeight) },
      }),
    );

    const heatData: GeoJSON.FeatureCollection<GeoJSON.Point, { weight: number }> = {
      type: 'FeatureCollection',
      features,
    };

    const source = this.map.getSource(HEAT_SOURCE_ID) as maplibregl.GeoJSONSource | undefined;
    if (source) {
      source.setData(heatData);
      return;
    }

    this.map.addSource(HEAT_SOURCE_ID, {
      type: 'geojson',
      data: heatData,
    });
  }

  private upsertHeatLayers(): void {
    if (!this.map) {
      return;
    }

    if (!this.map.getLayer(HEAT_LAYER_ID)) {
      this.map.addLayer({
        id: HEAT_LAYER_ID,
        type: 'heatmap',
        source: HEAT_SOURCE_ID,
        maxzoom: 20,
        paint: {
          'heatmap-weight': ['coalesce', ['get', 'weight'], 1],
          'heatmap-radius': this.radius(),
          'heatmap-intensity': 0.65,
          'heatmap-opacity': [
            'interpolate',
            ['linear'],
            ['zoom'],
            0,
            0.35,
            8,
            0.45,
            12,
            0.55,
            16,
            0.65,
          ],
          'heatmap-color': [...HEATMAP_COLOR_SCALE],
        },
      });
    }

    if (!this.map.getLayer(HEAT_SOFTENER_LAYER_ID)) {
      this.map.addLayer({
        id: HEAT_SOFTENER_LAYER_ID,
        type: 'circle',
        source: HEAT_SOURCE_ID,
        paint: {
          'circle-radius': this.radius(),
          'circle-color': 'rgba(0,0,0,0)',
          'circle-opacity': 0,
        },
      });
    }
  }

  private updateHeatmapPaint(): void {
    if (!this.map || !this.map.getLayer(HEAT_LAYER_ID)) {
      return;
    }

    const isTrace = this.onlyTrace();
    const radius = isTrace ? 1 : this.radius();
    const blur = isTrace ? 0 : this.blur();
    const effectiveRadius = isTrace ? radius : radius + blur * 0.75;

    this.map.setPaintProperty(HEAT_LAYER_ID, 'heatmap-radius', effectiveRadius);
    this.map.setPaintProperty(HEAT_LAYER_ID, 'heatmap-intensity', isTrace ? 1 : 0.65);
    this.map.setPaintProperty(
      HEAT_LAYER_ID,
      'heatmap-opacity',
      isTrace ? 1 : ['interpolate', ['linear'], ['zoom'], 0, 0.35, 8, 0.45, 12, 0.55, 16, 0.65],
    );
    this.map.setPaintProperty(
      HEAT_LAYER_ID,
      'heatmap-color',
      isTrace
        ? ['interpolate', ['linear'], ['heatmap-density'], 0, 'rgba(0,0,255,0)', 1, 'rgb(0,0,255)']
        : HEATMAP_COLOR_SCALE,
    );

    if (this.map.getLayer(HEAT_SOFTENER_LAYER_ID)) {
      this.map.setPaintProperty(HEAT_SOFTENER_LAYER_ID, 'circle-radius', radius + blur);
    }
  }

  private upsertMarkersSource(): void {
    if (!this.map) {
      return;
    }

    const markerData: GeoJSON.FeatureCollection = {
      type: 'FeatureCollection',
      features: this.markerFeatures,
    };

    const source = this.map.getSource(MARKERS_SOURCE_ID) as maplibregl.GeoJSONSource | undefined;
    if (source) {
      source.setData(markerData);
      return;
    }

    this.map.addSource(MARKERS_SOURCE_ID, {
      type: 'geojson',
      data: markerData,
      cluster: true,
      clusterMaxZoom: 14,
      clusterRadius: 50,
    });
  }

  private upsertMarkerLayers(): void {
    if (!this.map) {
      return;
    }

    if (!this.map.getLayer(CLUSTERS_LAYER_ID)) {
      this.map.addLayer({
        id: CLUSTERS_LAYER_ID,
        type: 'circle',
        source: MARKERS_SOURCE_ID,
        filter: ['has', 'point_count'],
        paint: {
          'circle-color': this.getClusterColorExpression(),
          'circle-radius': 14,
        },
      });
    } else {
      this.map.setPaintProperty(
        CLUSTERS_LAYER_ID,
        'circle-color',
        this.getClusterColorExpression(),
      );
    }

    if (!this.map.getLayer(CLUSTER_COUNT_LAYER_ID)) {
      this.map.addLayer({
        id: CLUSTER_COUNT_LAYER_ID,
        type: 'symbol',
        source: MARKERS_SOURCE_ID,
        filter: ['has', 'point_count'],
        layout: {
          'text-field': ['get', 'point_count_abbreviated'],
          'text-size': 13,
          'text-font': ['Open Sans Bold', 'Arial Unicode MS Bold'],
        },
        paint: {
          'text-color': '#ffffff',
          'text-halo-color': 'rgba(0,0,0,0.75)',
          'text-halo-width': 1.25,
          'text-halo-blur': 0.5,
        },
      });
    } else {
      this.map.setPaintProperty(CLUSTER_COUNT_LAYER_ID, 'text-color', '#ffffff');
      this.map.setPaintProperty(CLUSTER_COUNT_LAYER_ID, 'text-halo-color', 'rgba(0,0,0,0.75)');
      this.map.setPaintProperty(CLUSTER_COUNT_LAYER_ID, 'text-halo-width', 1.25);
      this.map.setPaintProperty(CLUSTER_COUNT_LAYER_ID, 'text-halo-blur', 0.5);
    }

    if (!this.map.getLayer(UNCLUSTERED_LAYER_ID)) {
      this.map.addLayer({
        id: UNCLUSTERED_LAYER_ID,
        type: 'circle',
        source: MARKERS_SOURCE_ID,
        filter: ['!', ['has', 'point_count']],
        paint: {
          'circle-color': '#ef4444',
          'circle-radius': 6,
          'circle-stroke-width': 1,
          'circle-stroke-color': '#ffffff',
        },
      });
    }
  }

  private bindMarkerInteractions(): void {
    if (!this.map || this.markerInteractionsBound) {
      return;
    }
    this.markerInteractionsBound = true;

    this.map.on('click', 'clusters', (event) => {
      this.zoomToCluster(event);
    });

    this.map.on('click', 'cluster-count', (event) => {
      this.zoomToCluster(event);
    });

    this.map.on('click', 'unclustered-point', (event) => {
      const feature = event.features?.[0];
      const rawPopupData = feature?.properties?.['popup_data'];
      const popupData = this.parsePopupData(rawPopupData);
      if (!popupData || !event.lngLat) {
        return;
      }

      const componentRef = createComponent(WorkoutPopup, {
        environmentInjector: this.environmentInjector,
      });
      componentRef.setInput('data', popupData);
      this.applicationRef.attachView(componentRef.hostView);

      const popupElement = componentRef.location.nativeElement as HTMLElement;

      this.markerPopup?.remove();
      this.markerPopup = new maplibregl.Popup({ closeButton: true })
        .setLngLat(event.lngLat)
        .setDOMContent(popupElement)
        .addTo(this.map!);

      this.markerPopup.once('close', () => {
        this.applicationRef.detachView(componentRef.hostView);
        componentRef.destroy();
      });
    });

    this.map.on('mouseenter', 'clusters', () => {
      this.map!.getCanvas().style.cursor = 'pointer';
    });

    this.map.on('mouseenter', 'cluster-count', () => {
      this.map!.getCanvas().style.cursor = 'pointer';
    });

    this.map.on('mouseleave', 'clusters', () => {
      this.map!.getCanvas().style.cursor = '';
    });

    this.map.on('mouseleave', 'cluster-count', () => {
      this.map!.getCanvas().style.cursor = '';
    });

    this.map.on('mouseenter', 'unclustered-point', () => {
      this.map!.getCanvas().style.cursor = 'pointer';
    });

    this.map.on('mouseleave', 'unclustered-point', () => {
      this.map!.getCanvas().style.cursor = '';
    });
  }

  private parsePopupData(rawPopupData: unknown): WorkoutPopupData | null {
    if (!rawPopupData) {
      return null;
    }

    if (typeof rawPopupData === 'string') {
      try {
        return JSON.parse(rawPopupData) as WorkoutPopupData;
      } catch (err) {
        console.error('Failed to parse popup data:', err);
        return null;
      }
    }

    if (typeof rawPopupData === 'object') {
      return rawPopupData as WorkoutPopupData;
    }

    return null;
  }

  private zoomToCluster(event: maplibregl.MapLayerMouseEvent): void {
    if (!this.map) {
      return;
    }

    const feature = event.features?.[0];
    const clusterId = feature?.properties?.['cluster_id'];

    if (clusterId === undefined || !event.lngLat) {
      return;
    }

    const source = this.map.getSource(MARKERS_SOURCE_ID) as maplibregl.GeoJSONSource | undefined;
    source
      ?.getClusterExpansionZoom(clusterId)
      .then((zoom) => {
        this.map?.easeTo({ center: event.lngLat, zoom });
      })
      .catch((err: unknown) => {
        console.error('Failed to get cluster expansion zoom:', err);
      });
  }

  private setMarkerVisibility(visible: boolean): void {
    if (!this.map) {
      return;
    }

    const visibility = visible ? 'visible' : 'none';

    if (this.map.getLayer(CLUSTERS_LAYER_ID)) {
      this.map.setLayoutProperty(CLUSTERS_LAYER_ID, 'visibility', visibility);
    }
    if (this.map.getLayer(CLUSTER_COUNT_LAYER_ID)) {
      this.map.setLayoutProperty(CLUSTER_COUNT_LAYER_ID, 'visibility', visibility);
    }
    if (this.map.getLayer(UNCLUSTERED_LAYER_ID)) {
      this.map.setLayoutProperty(UNCLUSTERED_LAYER_ID, 'visibility', visibility);
    }
  }

  private fitInitialBounds(): void {
    if (!this.map) {
      return;
    }

    const bounds = new LngLatBounds();
    for (const [lat, lng] of this.heatMapData) {
      bounds.extend([lng, lat]);
    }

    for (const feature of this.markerFeatures) {
      if (feature.geometry.type === 'Point') {
        const [lng, lat] = feature.geometry.coordinates;
        bounds.extend([lng, lat]);
      }
    }

    if (!bounds.isEmpty()) {
      this.map.fitBounds(bounds, { padding: 30, animate: false });
    }
  }

  protected refreshAfterStyleChange(): void {
    this.refreshHeatmapAfterStyleChange();
  }

  private readonly onMapMoveEnd = (): void => {
    void this.refreshCoordinatesIfNeeded();
  };

  private refreshHeatmapAfterStyleChange(): void {
    if (!this.map) {
      return;
    }

    const version = ++this.styleRefreshVersion;

    const tryRender = (): void => {
      if (!this.map || version !== this.styleRefreshVersion) {
        return;
      }
      if (this.map.isStyleLoaded()) {
        this.rerenderHeatMap();
        return;
      }
      window.setTimeout(tryRender, 50);
    };

    window.setTimeout(tryRender, 0);
  }

  private getRequestedCellSize(): number | undefined {
    if (this.onlyTrace()) {
      return undefined;
    }

    const zoom = this.map?.getZoom() ?? 0;
    if (zoom >= 11) {
      return FINE_HEATMAP_CELL_SIZE;
    }
    if (zoom >= 8) {
      return MEDIUM_HEATMAP_CELL_SIZE;
    }
    return DEFAULT_HEATMAP_CELL_SIZE;
  }

  private async refreshCoordinatesIfNeeded(
    force = false,
    includeViewportBounds = true,
  ): Promise<void> {
    if (!this.map) {
      return;
    }

    const viewport = includeViewportBounds ? this.getSanitizedViewportBounds() : undefined;
    const cellSize = this.getRequestedCellSize();
    const requestKey = JSON.stringify({
      mode: cellSize === undefined ? 'raw' : Number(cellSize.toFixed(6)),
      bounds: viewport
        ? {
            minLat: Number(viewport.minLat.toFixed(4)),
            minLng: Number(viewport.minLng.toFixed(4)),
            maxLat: Number(viewport.maxLat.toFixed(4)),
            maxLng: Number(viewport.maxLng.toFixed(4)),
          }
        : 'global',
    });
    if (!force && requestKey === this.coordinatesRequestKey) {
      return;
    }

    this.coordinatesRequestKey = requestKey;
    const version = ++this.coordinatesRefreshVersion;

    try {
      const coordinatesResponse = await firstValueFrom(
        this.api.getWorkoutsCoordinates({
          cellSize,
          minLat: viewport?.minLat,
          minLng: viewport?.minLng,
          maxLat: viewport?.maxLat,
          maxLng: viewport?.maxLng,
        }),
      );
      if (version !== this.coordinatesRefreshVersion) {
        return;
      }
      this.heatMapData = coordinatesResponse?.results ?? [];
      this.maxPointCount = this.computeMaxPointCount(this.markerFeatures);
      this.rerenderHeatMap();
    } catch (err) {
      console.error('Failed to load heatmap coordinates:', err);
    }
  }

  private getSanitizedViewportBounds():
    | { minLat: number; minLng: number; maxLat: number; maxLng: number }
    | undefined {
    if (!this.map) {
      return undefined;
    }

    const bounds = this.map.getBounds();
    const south = bounds.getSouth();
    const north = bounds.getNorth();
    const west = bounds.getWest();
    const east = bounds.getEast();

    if (![south, north, west, east].every(Number.isFinite)) {
      return undefined;
    }

    // When the viewport spans the wrapped world, a single min/max lng range is not valid.
    if (east - west >= 360) {
      return undefined;
    }

    const minLat = Math.max(-90, Math.min(90, south));
    const maxLat = Math.max(-90, Math.min(90, north));
    const minLng = this.normalizeLongitude(west);
    const maxLng = this.normalizeLongitude(east);

    // Crossing the antimeridian cannot be represented by one [min,max] range for this endpoint.
    if (minLng > maxLng || minLat > maxLat) {
      return undefined;
    }

    return { minLat, minLng, maxLat, maxLng };
  }

  private normalizeLongitude(value: number): number {
    return ((((value + 180) % 360) + 360) % 360) - 180;
  }

  private getClusterColorExpression(): ExpressionSpecification {
    return [
      'interpolate',
      ['linear'],
      ['/', ['to-number', ['get', 'point_count'], 0], this.maxPointCount],
      0,
      'rgb(0,0,255)',
      0.5,
      'rgb(0,255,0)',
      0.75,
      'rgb(255,255,0)',
      1,
      'rgb(255,0,0)',
    ];
  }

  private computeMaxPointCount(features: GeoJSON.Feature[]): number {
    const fallback = Math.max(features.length, 1);

    return features.reduce((max, feature) => {
      const pointCount = feature.properties?.['point_count'];
      if (typeof pointCount !== 'number' || !Number.isFinite(pointCount)) {
        return max;
      }
      return Math.max(max, pointCount);
    }, fallback);
  }
}
