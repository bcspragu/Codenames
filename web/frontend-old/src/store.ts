import axios from 'axios';
import { State, User } from './state';

class Store {
  public state: State;

  constructor(state: State) {
    this.state = state;
  }

  public setUser(user: User) {
    this.state.user = user;
  }

  public loadUser(): Promise<User> {
    return axios.get('/api/user').then((resp) => {
      this.setUser(resp.data);
      return resp.data;
    });
  }

  public getUser(): Promise<User> {
    if (this.state.user) {
      return Promise.resolve(this.state.user);
    }
    return this.loadUser();
  }
}

const store: Store = new Store(new State());

export default store;
