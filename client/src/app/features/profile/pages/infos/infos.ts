import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { TranslatePipe } from '@ngx-translate/core';
import { ProfileStore } from '../../services/profile-store';

@Component({
  selector: 'app-profile-infos',
  imports: [ReactiveFormsModule, TranslatePipe],
  templateUrl: './infos.html',
  styleUrl: './infos.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ProfileInfosPage {
  protected store = inject(ProfileStore);
}
