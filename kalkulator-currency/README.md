Konsep penting yang dipelajari pada proyek ini :
1. http.HandleFunc — mendaftarkan fungsi ke sebuah path URL. Setiap kali ada request ke path itu, fungsi tersebut dipanggil.
2. r.URL.Query().Get("key") — cara membaca query parameter dari URL seperti ?from=USD&to=IDR.
3. r.Method — mengecek apakah request adalah GET, POST, PUT, atau DELETE.
4. w.Header().Set(...) — mengatur header response sebelum menulis body. Header harus di-set sebelum w.WriteHeader().
5. json.NewEncoder(w).Encode(data) — menulis struct Go langsung ke response sebagai JSON.
6. w.WriteHeader(status) — mengirim HTTP status code (200, 400, 404, dst). Kalau tidak dipanggil, default-nya 200.
