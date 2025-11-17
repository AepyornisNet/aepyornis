import { ChangeDetectionStrategy, Component, effect, ElementRef, input, OnDestroy, viewChild, ViewEncapsulation } from '@angular/core';
import { CommonModule } from '@angular/common';
import * as L from 'leaflet';
import { MapPoint } from '../../../../core/types/route-segment';

@Component({
  selector: 'app-route-segment-map',
  imports: [CommonModule],
  templateUrl: './route-segment-map.html',
  styleUrls: ['./route-segment-map.scss'],
  encapsulation: ViewEncapsulation.None,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RouteSegmentMapComponent implements OnDestroy {
  private readonly mapContainer = viewChild<ElementRef<HTMLDivElement>>('mapContainer');

  public readonly points = input<MapPoint[] | null>(null);
  // Optional selection for highlighting (0-based indices inclusive)
  public readonly selection = input<{ startIndex: number; endIndex: number } | null>(null);
  public readonly center = input<{ lat: number; lng: number } | null>(null);

  private map?: L.Map;
  private baseTrackLayer?: L.FeatureGroup;
  private highlightLayer?: L.FeatureGroup;
  private path?: [number, number][];

  public constructor() {
    // Initialize map when points arrive
    effect(() => {
      const pts = this.points();
      const containerRef = this.mapContainer();
      if (!this.map && containerRef && pts && pts.length > 1) {
        this.initMap();
      }
    });

    // React to selection changes
    effect(() => {
      const sel = this.selection();
      this.highlightSelection(sel);
    });
  }

  public ngOnDestroy(): void {
    if (this.map) {
      this.map.remove();
    }
  }

  private initMap(): void {
    const containerRef = this.mapContainer();
    const pts = this.points();
    if (!containerRef || !pts || pts.length < 2) {
      return;
    }

    const mid = Math.floor(pts.length / 2);
    const centerLat = this.center()?.lat ?? pts[mid].lat;
    const centerLng = this.center()?.lng ?? pts[mid].lng;

    this.map = L.map(containerRef.nativeElement, {
      fadeAnimation: false,
      preferCanvas: true,
      renderer: L.canvas()
    }).setView([centerLat, centerLng], 15);

    const streetLayer = L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '&copy; OpenStreetMap contributors',
      className: 'map-tiles',
    }).addTo(this.map);
    const aerialLayer = L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}', {
      attribution: 'Esri',
    });

    this.baseTrackLayer = L.featureGroup().addTo(this.map);

    // Draw base track as single green polyline (no elevation coloring)
    this.path = pts.map(p => [p.lat, p.lng]);
    L.polyline(this.path, { color: 'green', weight: 4, opacity: 0.9 }).addTo(this.baseTrackLayer);

    // Start/end markers
    const first = pts[0];
    const last = pts[pts.length - 1];
    L.circleMarker([first.lat, first.lng], { color: 'green', radius: 6, fillOpacity: 1 }).addTo(this.baseTrackLayer);
    L.circleMarker([last.lat, last.lng], { color: 'red', radius: 6, fillOpacity: 1 }).addTo(this.baseTrackLayer);

    // Layer control
    L.control.layers({ Streets: streetLayer, Aerial: aerialLayer }, {}, { position: 'topright' }).addTo(this.map);

    this.map.fitBounds(this.baseTrackLayer.getBounds(), { animate: false });
    L.control.scale().addTo(this.map);
  }

  private highlightSelection(sel: { startIndex: number; endIndex: number } | null): void {
    if (!this.map || !this.points()) {
      return;
    }
    // Clear previous highlight
    if (this.highlightLayer) {
      this.highlightLayer.remove();
      this.highlightLayer = undefined;
    }
    if (!sel) {
      // Reset zoom to full track if cleared
      if (this.baseTrackLayer) {
        this.map.fitBounds(this.baseTrackLayer.getBounds(), { animate: false });
      }
      return;
    }
    const pts = this.points()!;
    const start = Math.max(0, Math.min(sel.startIndex, pts.length - 2));
    const end = Math.max(start + 1, Math.min(sel.endIndex, pts.length - 1));

    // Draw single polyline for entire selection (much faster than per-segment)
    this.highlightLayer = L.featureGroup().addTo(this.map);
    const highlightLatLngs = this.path?.slice(start, end + 1) ?? pts.slice(start, end + 1).map(p => [p.lat, p.lng] as [number, number]);
    L.polyline(highlightLatLngs, { color: 'red', weight: 5, opacity: 0.8 }).addTo(this.highlightLayer);
    this.map.fitBounds(this.highlightLayer.getBounds(), { animate: false });
  }
}
