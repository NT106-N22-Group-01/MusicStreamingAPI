### Authentication

Khi máy chủ được mở, bạn không cần xác thực các yêu cầu đến API. Các cài đặt được bảo vệ bằng username và password yêu cầu bạn xác thực các yêu cầu khi sử dụng API. Để làm điều này, hỗ trợ các phương thức sau đây:

* Bearer token trong HTTP header `Authorization` (như mô tả trong [RFC 6750](https://tools.ietf.org/html/rfc6750)):

```
Authorization: Bearer {token}

```

Authentication tokens được nhận thông qua `/v1/login/token/` hoặc `/v1/register/token/` endpoint được mô tả dưới đây. Sử dụng tokens là phương pháp được ưu tiên vì nó không tiết lộ username và password ở mỗi request.

### Endpoints

* [Search](#search)
* [Browse](#browse)
* [Play a Song](#play-a-song)
* [ListenCount](#count-a-song)
* [Download an Album](#download-an-album)
* [Album Artwork](#album-artwork)
  * [Get Artwork](#get-artwork)
* [Artist Image](#artist-image)
  * [Get Artist Image](#get-artist-image)
* [Token Request](#token-request)
* [Register Token](#register-token)

### Search

Thự hiện tìm kiếm thông qua endpoint sau:

```sh
GET /v1/search/?q={query}
```

nó sẽ trả về mảng JSON chưa thông tin các track. Mỗi đối tượng trong JSON đại diện cho một track duy nhất khớp với `query` được cung cấp. Ví dụ:

```js
[
    {
        "id": 22,
        "artist_id": 6,
        "artist": "Aimer",
        "album_id": 3,
        "album": "春はゆく / marie",
        "title": "春はゆく",
        "track": 1,
        "format": "flac",
        "duration": 304000
    },
    {
        "id": 23,
        "artist_id": 6,
        "artist": "Aimer",
        "album_id": 3,
        "album": "春はゆく / marie",
        "title": "marie",
        "track": 2,
        "format": "flac",
        "duration": 307000
    }
]
```

### Browse

Cách để duyệt toàn bộ bộ sưu tập là thông qua gọi API `browse`. Nó cho phép bạn lấy các album hoặc nghệ sĩ trong một trình tự được sắp xếp và phân trang.

```sh
GET /v1/browse/[?by=artist|album][&per-page={number}][&page={number}][&order-by=id|name][&order=desc|asc]
```

JSON trả về chứa dữ liệu cho trang hiện tại, số trang trong tất cả các trang cho phương thức duyệt hiện tại và các URL của trang tiếp theo hoặc trang trước đó.

```js
{
    "data": [
        {
            "album_id": 1,
            "album": "102112",
            "artist": "Cá Hồi Hoang"
        },
        {
            "album_id": 2,
            "album": "Avid / Hands Up to the Sky",
            "artist": "Various Artists"
        },
        {
            "album_id": 3,
            "album": "春はゆく / marie",
            "artist": "Aimer"
        }
    ],
    "next": "",
    "previous": "",
    "pages_count": 1
}
```

Hiện tại, có hai giá trị có thể cho tham số `by`. Do đó, có hai loại data có thể được trả về: "artist" (nghệ sĩ) và "album" (đây là giá trị **mặc định**).

**by=artist**

kết quả sẽ có các giá trị như sau:

```js
{
  "artist": "Jefferson Airplane",
  "artist_id": 73
}
```

**by=album**

kết quả sẽ có các giá trị như sau:

```js
{
  "album": "Battlefield Vietnam"
  "artist": "Jefferson Airplane",
  "album_id": 2
}
```

**Các tham số bổ sung:**

_per-page_: điều khiển số lượng mục sẽ có trong trường `data` cho từng trang cụ thể. Giá trị **mặc định là 10**.

_page_: dữ liệu được tạo sẽ là cho trang này. **Giá trị mặc định là 1**.

_order-by_: điều khiển cách kết quả sẽ được sắp xếp. Giá trị id có nghĩa là sắp xếp sẽ được thực hiện theo ID của album hoặc nghệ sĩ, tùy thuộc vào đối số by. Tương tự, điều này cũng áp dụng cho giá trị `name`. **Mặc định là `name`**.

_order_: điều khiển xem thứ tự sẽ tăng dần (giá trị `asc`) hay giảm dần (giá trị `desc`). **Mặc định là `asc`**.

### Phát nhạc

```
GET /v1/file/{trackID}
```

Endpoint này sẽ trả về tập tin nhạc. `trackID` của một bài hát có thể được tìm thấy bằng cuộc gọi API tìm kiếm.

### Lượt nghe

```
GET /v1/file/{trackID}/count
```

Endpoint này sẽ trả về số lượt nghe của track có ID là trackID.

### Tải Album

```
GET /v1/album/{albumID}
```

Endpoint này sẽ trả về tập tin nén trong đó chứa toàn bộ nhạc của album này.

### Album Artwork


#### Get Artwork

```
GET /v1/album/{albumID}/artwork
```

Trả về bitmap image là artwork cho album này nếu có sẵn. Tìm kiếm artwork hoạt động như sau: thư mục của album sẽ được quét để tìm bất kỳ hình ảnh nào (tệp png/jpeg/gif/tiff), và nếu có bất kỳ hình ảnh nào trông giống hình minh hoạ, nó sẽ được hiển thị. Nếu thất bại, bạn có thể config để tìm kiếm trong [MusicBrainz Cover Art Archive](https://musicbrainz.org/doc/Cover_Art_Archive/). Mặc định, không có external calls được thực hiện.

Mặc định, hình ảnh kích thước đầy đủ sẽ được phục vụ. Bạn có thể yêu cầu hình thu nhỏ bằng cách thêm truy vấn `?size=small`.

### Artist Image

#### Get Artist Image

```
GET /v1/artist/{artistID}/image
```

Trả về bitmap image là hình đại diện cho artist nếu có sẵn. Quá trình tìm kiếm hình ảnh hoạt động như sau: nếu hình ảnh của nghệ sĩ được tìm thấy trong cơ sở dữ liệu, nó sẽ được sử dụng. Trong trường hợp không tìm thấy, Server sẽ sử dụng các API MusicBrainz và Discogs để lấy hình ảnh.

Mặc định, hình ảnh kích thước đầy đủ sẽ được phục vụ. Bạn có thể yêu cầu một hình thu nhỏ bằng cách thêm truy vấn `?size=small`.

### Login

```
POST /v1/login/token/
{
  "username": "your-username",
  "password": "your-password"
}
```

Nếu username và password đúng Endpoint sẽ trả về token để thêm vào header phục vụ cho việc xác thực.

### Register 

```
POST /v1/register/token/
{
  "username": "your-username",
  "password": "your-password"
}
```

Endpoint này sẽ tạo tài khoản cho người dùng sau khi tạo thành công Endpoint sẽ trả về token để thêm vào header phục vụ cho việc xác thực.
