import { ChangeDetectionStrategy, Component, inject, OnInit } from '@angular/core';
import { RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { TranslatePipe } from '@ngx-translate/core';
import { ProfileStore } from '../../services/profile-store';

@Component({
  selector: 'app-profile',
  imports: [RouterOutlet, RouterLink, RouterLinkActive, TranslatePipe],
  templateUrl: './profile.html',
  styleUrl: './profile.scss',
  providers: [ProfileStore],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Profile implements OnInit {
  protected store = inject(ProfileStore);

  public readonly navigationItems = [
    { label: 'General', route: '/profile/general' },
    { label: 'Personal info', route: '/profile/infos' },
    { label: 'Privacy', route: '/profile/privacy' },
    { label: 'Followers', route: '/profile/followers' },
    { label: 'Actions', route: '/profile/actions' },
  ];

  public ngOnInit(): void {
    this.store.loadProfile();
  }
}
