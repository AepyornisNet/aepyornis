import { ChangeDetectionStrategy, Component, effect, input } from '@angular/core';

import { NgxMapLibreGLModule } from '@maplibre/ngx-maplibre-gl';
import maplibregl, { LngLatBounds, Map } from 'maplibre-gl';
import { MapPoint } from '../../../../core/types/route-segment';
import { BaseMapComponent } from '../../../../core/components/base-map/base-map';

@Component({
  selector: 'app-route-segment-map',
  imports: [NgxMapLibreGLModule],
  templateUrl: './route-segment-map.html',
  styleUrls: ['./route-segment-map.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RouteSegmentMapComponent extends BaseMapComponent {
  public readonly points = input<MapPoint[] | null>(null);
  public readonly selection = input<{ startIndex: number; endIndex: number } | null>(null);
  public readonly center = input<{ lat: number; lng: number } | null>(null);

  private startMarker?: maplibregl.Marker;
  private endMarker?: maplibregl.Marker;

  public constructor() {
    super();
    effect(() => {
      this.points();
      if (this.map && this.map.isStyleLoaded()) {
        this.renderTrack();
      }
    });

    effect(() => {
      this.highlightSelection(this.selection());
    });
  }

  public onMapLoad(map: Map): void {
    this.onMapLoadBase(map);
    this.refreshTrackAfterStyleChange();
  }

  private renderTrack(): void {
    const pts = this.points();
    if (!this.map || !pts || pts.length < 2) {
      return;
    }

    this.clearTrack();

    const coordinates = pts.map((p) => [p.lng, p.lat] as [number, number]);

    this.map.addSource('route-track-source', {
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
      id: 'route-track-layer',
      type: 'line',
      source: 'route-track-source',
      paint: {
        'line-color': 'green',
        'line-width': ['interpolate', ['linear'], ['zoom'], 0, 2, 10, 3, 15, 4],
        'line-opacity': 0.9,
      },
    });

    this.startMarker = new maplibregl.Marker({ color: 'green', scale: 0.8 })
      .setLngLat([pts[0].lng, pts[0].lat])
      .addTo(this.map);

    this.endMarker = new maplibregl.Marker({ color: 'red', scale: 0.8 })
      .setLngLat([pts[pts.length - 1].lng, pts[pts.length - 1].lat])
      .addTo(this.map);

    this.fitToCoordinates(coordinates);
    this.highlightSelection(this.selection());
  }

  private highlightSelection(sel: { startIndex: number; endIndex: number } | null): void {
    if (!this.map || !this.points()) {
      return;
    }

    if (this.map.getLayer('route-highlight-layer')) {
      this.map.removeLayer('route-highlight-layer');
    }
    if (this.map.getSource('route-highlight-source')) {
      this.map.removeSource('route-highlight-source');
    }

    if (!sel) {
      const coords = this.points()!.map((p) => [p.lng, p.lat] as [number, number]);
      this.fitToCoordinates(coords);
      return;
    }

    const pts = this.points()!;
    const start = Math.max(0, Math.min(sel.startIndex, pts.length - 2));
    const end = Math.max(start + 1, Math.min(sel.endIndex, pts.length - 1));
    const coords = pts.slice(start, end + 1).map((p) => [p.lng, p.lat] as [number, number]);

    this.map.addSource('route-highlight-source', {
      type: 'geojson',
      data: {
        type: 'FeatureCollection',
        features: [
          {
            type: 'Feature',
            geometry: { type: 'LineString', coordinates: coords },
            properties: {},
          },
        ],
      },
    });

    this.map.addLayer({
      id: 'route-highlight-layer',
      type: 'line',
      source: 'route-highlight-source',
      paint: {
        'line-color': 'red',
        'line-width': ['interpolate', ['linear'], ['zoom'], 0, 3, 10, 4, 15, 5],
        'line-opacity': 0.8,
      },
    });

    this.fitToCoordinates(coords);
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
        this.renderTrack();
        return;
      }
      window.setTimeout(tryRender, 50);
    };

    window.setTimeout(tryRender, 0);
  }

  private fitToCoordinates(coords: [number, number][]): void {
    if (!this.map || coords.length === 0) {
      return;
    }

    const bounds = new LngLatBounds(coords[0], coords[0]);
    for (const coord of coords) {
      bounds.extend(coord);
    }

    this.map.fitBounds(bounds, { padding: 25, animate: false });
  }

  private clearTrack(): void {
    if (!this.map) {
      return;
    }

    const layers = ['route-highlight-layer', 'route-track-layer'];
    const sources = ['route-highlight-source', 'route-track-source'];

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
  }
}
