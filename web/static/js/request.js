/**
 * 请求封装
 * @param url
 * @param method
 * @param data
 * @param header
 * @returns {Promise<unknown>}
 */
const baseRequest = (url, method,  data , header) => {
    return new Promise((resolve, reject) => {
        fetch(url, {
            method: method,
            headers: header,
            body: data
        }).then(res => res.json()).then(data => {
            resolve(data);
        }).catch((err) => {
            reject(err);
        })
    })
}