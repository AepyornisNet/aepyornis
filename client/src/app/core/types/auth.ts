export const AUTH_LOGIN_URL = '/api/v2/auth/signin';
export const AUTH_REGISTER_URL = '/api/v2/auth/register';
export const AUTH_LOGOUT_URL = '/api/v2/auth/signout';

export type SignInRequest = {
	username: string;
	password: string;
};

export type RegisterRequest = {
	username: string;
	password: string;
	name?: string;
};
