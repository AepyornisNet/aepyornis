import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth-guard';
import { adminGuard } from './core/guards/admin-guard';

export const routes: Routes = [
  { path: '', redirectTo: '/dashboard', pathMatch: 'full' },
  {
    path: 'login',
    loadComponent: () => import('./features/auth/pages/login/login').then((m) => m.Login),
  },
  {
    path: '',
    loadComponent: () =>
      import('./layouts/authenticated-layout/authenticated-layout').then(
        (m) => m.AuthenticatedLayout,
      ),
    canActivate: [authGuard],
    children: [
      {
        path: 'dashboard',
        loadComponent: () =>
          import('./features/dashboard/pages/dashboard/dashboard').then((m) => m.Dashboard),
      },
      {
        path: 'workouts',
        children: [
          {
            path: '',
            loadComponent: () =>
              import('./features/workouts/pages/list/workouts').then((m) => m.Workouts),
          },
          {
            path: 'add',
            loadComponent: () =>
              import('./features/workouts/pages/create/workout-create').then(
                (m) => m.WorkoutCreate,
              ),
          },
          {
            path: ':id',
            loadComponent: () =>
              import('./features/workouts/pages/detail/workout-detail').then(
                (m) => m.WorkoutDetailPage,
              ),
          },
          {
            path: ':id/edit',
            loadComponent: () =>
              import('./features/workouts/pages/create/workout-create').then(
                (m) => m.WorkoutCreate,
              ),
          },
          {
            path: ':id/create-route-segment',
            loadComponent: () =>
              import(
                './features/route-segments/pages/create-workout/create-workout-route-segment'
              ).then((m) => m.CreateWorkoutRouteSegmentPage),
          },
        ],
      },
      {
        path: 'measurements',
        loadComponent: () =>
          import('./features/measurements/pages/measurements/measurements').then(
            (m) => m.Measurements,
          ),
      },
      {
        path: 'statistics',
        loadComponent: () =>
          import('./features/statistics/pages/statistics/statistics').then((m) => m.Statistics),
      },
      {
        path: 'statistics/records/climbs/:workoutType',
        loadComponent: () =>
          import('./features/statistics/pages/climb-ranking/climb-ranking').then(
            (m) => m.ClimbRankingPage,
          ),
      },
      {
        path: 'statistics/records/:workoutType/:label',
        loadComponent: () =>
          import('./features/statistics/pages/records-ranking/records-ranking').then(
            (m) => m.RecordsRankingPage,
          ),
      },
      {
        path: 'statistics/records',
        loadComponent: () =>
          import('./features/statistics/pages/records/records').then((m) => m.StatisticsRecords),
      },
      {
        path: 'heatmap',
        loadComponent: () =>
          import('./features/statistics/pages/heatmap/heatmap').then((m) => m.Heatmap),
      },
      {
        path: 'route-segments',
        children: [
          {
            path: '',
            loadComponent: () =>
              import('./features/route-segments/pages/list/route-segments').then(
                (m) => m.RouteSegments,
              ),
          },
          {
            path: 'create',
            loadComponent: () =>
              import('./features/route-segments/pages/create/create-route-segment').then(
                (m) => m.CreateRouteSegmentPage,
              ),
          },
          {
            path: ':id',
            loadComponent: () =>
              import('./features/route-segments/pages/detail/route-segment-detail').then(
                (m) => m.RouteSegmentDetailPage,
              ),
          },
          {
            path: ':id/edit',
            loadComponent: () =>
              import('./features/route-segments/pages/edit/edit-route-segment').then(
                (m) => m.EditRouteSegment,
              ),
          },
        ],
      },
      {
        path: 'profile',
        loadComponent: () =>
          import('./features/profile/pages/profile/profile').then((m) => m.Profile),
        children: [
          {
            path: '',
            pathMatch: 'full',
            redirectTo: 'general',
          },
          {
            path: 'general',
            loadComponent: () =>
              import('./features/profile/pages/general/general').then(
                (m) => m.ProfileGeneralPage,
              ),
          },
          {
            path: 'infos',
            loadComponent: () =>
              import('./features/profile/pages/infos/infos').then((m) => m.ProfileInfosPage),
          },
          {
            path: 'privacy',
            loadComponent: () =>
              import('./features/profile/pages/privacy/privacy').then((m) => m.ProfilePrivacyPage),
          },
          {
            path: 'followers',
            loadComponent: () =>
              import('./features/profile/pages/followers/followers').then(
                (m) => m.ProfileFollowersPage,
              ),
          },
          {
            path: 'actions',
            loadComponent: () =>
              import('./features/profile/pages/actions/actions').then(
                (m) => m.ProfileActionsPage,
              ),
          },
        ],
      },
      {
        path: 'admin',
        loadComponent: () => import('./features/admin/pages/admin/admin').then((m) => m.Admin),
        canActivate: [adminGuard],
      },
      {
        path: 'admin/users/:id/edit',
        loadComponent: () =>
          import('./features/admin/pages/user-edit/user-edit').then((m) => m.UserEdit),
        canActivate: [adminGuard],
      },
      {
        path: 'equipment',
        children: [
          {
            path: '',
            loadComponent: () =>
              import('./features/equipment/pages/list/equipment').then((m) => m.Equipment),
          },
          {
            path: ':id',
            loadComponent: () =>
              import('./features/equipment/pages/detail/equipment-detail').then(
                (m) => m.EquipmentDetail,
              ),
          },
        ],
      },
    ],
  },
  { path: '**', redirectTo: '/dashboard' },
];
